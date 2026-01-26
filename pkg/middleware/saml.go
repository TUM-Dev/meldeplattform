package middleware

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"net/http"
	"net/url"
	"time"
)

var keys tls.Certificate

type SamlConfig struct {
	IdpMetadataURL string `yaml:"idpMetadataURL"`
	EntityID       string `yaml:"entityID"`
	RootURL        string `yaml:"rootURL"`
	Cert           struct {
		Org           string `yaml:"org"`
		Country       string `yaml:"country"`
		Province      string `yaml:"province"`
		Locality      string `yaml:"locality"`
		StreetAddress string `yaml:"streetAddress"`
		PostalCode    string `yaml:"postalCode"`
		Cn            string `yaml:"cn"`
	} `yaml:"cert"`
}

// secureCookies determines if cookies should have the Secure flag set
var secureCookies = false

// SetSecureCookies sets whether cookies should be secure (for production mode)
func SetSecureCookies(secure bool) {
	secureCookies = secure
}

func ConfigSaml(r *gin.Engine, c SamlConfig) {
	err := generateKey(c.Cert.Org, c.Cert.Country, c.Cert.Province, c.Cert.Locality, c.Cert.StreetAddress, c.Cert.PostalCode, c.Cert.Cn)
	if err != nil {
		fmt.Println("Skipping saml due to cert error:", err)
		return
	}
	// create saml.ServiceProvider
	keyPair, err := tls.LoadX509KeyPair("files/cert.pem", "files/key.pem")
	if err != nil {
		fmt.Println("Could not load SAML keypair", err)
		return
	}
	keys = keyPair
	keys.Leaf, err = x509.ParseCertificate(keys.Certificate[0])
	if err != nil {
		fmt.Println("Could not parse SAML keypair", err)
		return
	}
	idpMetadataURL, err := url.Parse(c.IdpMetadataURL)
	if err != nil {
		fmt.Println("Could not parse Identity Provider metadata URL", err)
		return
	}
	idpMetadata, err := samlsp.FetchMetadata(context.Background(), http.DefaultClient,
		*idpMetadataURL)
	if err != nil {
		fmt.Println("Could not load Identity Provider metadata", err)
		return
	}

	rootURL, err := url.Parse(c.RootURL)
	if err != nil {
		fmt.Println("Could not parse Root URL", err)
		return
	}

	samlSP, err := samlsp.New(samlsp.Options{
		URL:               *rootURL,
		Key:               keys.PrivateKey.(*rsa.PrivateKey),
		Certificate:       keys.Leaf,
		IDPMetadata:       idpMetadata,
		EntityID:          c.EntityID,
		AllowIDPInitiated: true,
	})
	if err != nil {
		fmt.Println("Could not create SAML Service Provider", err)
	}
	samlSP.ServiceProvider.AcsURL = *rootURL

	// serve metadata. This can be fetched periodically by the IDP.
	r.GET("/saml/metadata", func(c *gin.Context) {
		samlSP.ServeMetadata(c.Writer, c.Request)
	})

	// /saml/out is accessed to log in with the IDP.
	// It will redirect to http://login.idp.something/... which will redirect back to us on success.
	r.GET("/saml/out", func(c *gin.Context) {
		samlSP.HandleStartAuthFlow(c.Writer, c.Request)
	})

	// /saml/slo is accessed after the IDP logged out the user.
	r.POST("/saml/slo", func(c *gin.Context) {
		err := c.Request.ParseForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		err = samlSP.ServiceProvider.ValidateLogoutResponseForm(c.Request.PostFormValue("SAMLResponse"))
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"code": "403- Forbidden", "error": "Invalid logout data: " + err.Error()})
			return
		}
		c.SetCookie("jwt", "", -1, "/", "", true, true)
		c.Redirect(http.StatusFound, "/")
	})

	// /saml/logout redirects to the idp with a logout request.
	// The idp will redirect back to /saml/slo after the user logged out.
	r.GET("/saml/logout", func(c *gin.Context) {
		c.SetCookie("jwt", "", -1, "/", "", true, true)
		c.Redirect(http.StatusFound, "/")
	})

	// /shib is accessed after authentication with the IDP. The post body contains the encrypted SAMLResponse.
	r.POST("/shib", func(c *gin.Context) {
		err := c.Request.ParseForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "400 - Bad Request", "error": err.Error()})
			return
		}
		response, err := samlSP.ServiceProvider.ParseResponse(c.Request, []string{""})
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"code": "403- Forbidden", "error": err.Error()})
			return
		}

		uid := extractSamlField(response, "uid")
		name := extractSamlField(response, "displayName")
		mail := extractSamlField(response, "mail")
		t := jwt.New(jwt.GetSigningMethod("RS256"))

		t.Claims = &TokenClaims{
			RegisteredClaims: &jwt.RegisteredClaims{
				ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Hour * 24 * 7)}, // Token expires in one week
			},
			Name: name,
			Mail: mail,
			UID:  uid,
		}
		signedString, err := t.SignedString(keys.PrivateKey.(*rsa.PrivateKey))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.SetCookie("jwt", signedString, 60*60*24*7, "/", "", secureCookies, true)
		c.Redirect(http.StatusFound, "/")
	})
}

type TokenClaims struct {
	*jwt.RegisteredClaims

	UID  string `json:"UID"`
	Name string `json:"name"`
	Mail string `json:"mail"`
}

// extractSamlField gets the value of the given field from the SAML response or an empty string if the field is not present.
func extractSamlField(assertion *saml.Assertion, friendlyFieldName string) string {
	for _, statement := range assertion.AttributeStatements {
		for _, attribute := range statement.Attributes {
			if attribute.FriendlyName == friendlyFieldName && len(attribute.Values) > 0 {
				return attribute.Values[0].Value
			}
		}
	}
	return ""
}
