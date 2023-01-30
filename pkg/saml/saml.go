package saml

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
)

type SamlConfig struct {
	IdpMetadataURL string `yaml:"idpMetadataURL"`
	EntityID       string `yaml:"entityID"`
	RootURL        string `yaml:"rootURL"`
}

func ConfigSaml(r *gin.Engine, c SamlConfig) {
	err := generateKey()
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
	keyPair.Leaf, err = x509.ParseCertificate(keyPair.Certificate[0])
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
		Key:               keyPair.PrivateKey.(*rsa.PrivateKey),
		Certificate:       keyPair.Leaf,
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
		c.SetCookie("jwt", "", -1, "/", "", false, true)
		c.Redirect(http.StatusFound, "/")
	})

	// /saml/logout redirects to the idp with a logout request.
	// The idp will redirect back to /saml/slo after the user logged out.
	r.GET("/saml/logout", func(c *gin.Context) {
		//todo
	})

	// /shib is accessed after authentication with the IDP. The post body contains the encrypted SAMLResponse.
	r.POST("/shib", func(c *gin.Context) {
		err := c.Request.ParseForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "400 - Bad Request", "error": err.Error()})
		}
		response, err := samlSP.ServiceProvider.ParseResponse(c.Request, []string{""})
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"code": "403- Forbidden", "error": err.Error()})
			return
		}

		lrzID := extractSamlField(response, "uid")
		fmt.Println("LRZ ID: ", lrzID)
	})
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
