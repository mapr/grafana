package login
/****************************************************************
** auth.go - is a module that ships with Grafana and this version
** of the code is a modify version for MapR so that Grafaha
** authenticates against the CLDB to gain PAM support which is
** consistent with the rest of MapR offerings.
******************************************************************/
import (
	"errors"
        "time"
        "math/rand"
        "strconv"
        "strings"
        "os"
        "bufio"
        "fmt"
	"crypto/subtle"
	"github.com/grafana/grafana/pkg/bus"
	m "github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/util"
        "bytes"
        "encoding/json"
        "io/ioutil"
        "net"
        "net/http"
        "crypto/tls"
        "crypto/x509"
)

var (
	ErrInvalidCredentials = errors.New("Invalid Username or Password")
	ErrCLDBConnectionFailed = errors.New("Failed to get response from CLDB")
)


type LoginUserQuery struct {
	Username string
	Password string
	User     *m.User
}
/******************************************************
** Following are specific for MapR in order to call
** the CLDB to verify User/pass
*******************************************************/
type JSONtoCLDB struct {
     Class string        `json:"class"`
     UserName string     `json:"userName"`
     PassWord string     `json:"passWord"`
     TicketDurInSecs int `json:"ticketDurInSecs"`
}
type JSONfromCLDB struct {
     Class string               `json:"class"`
     Error string               `json:"error"`
     Status int                 `json:"status"`
     TicketAndKeyString string  `json:"ticketAndKeyString"`
     Token string               `json:"token"`
}
type listCLDB struct {
     host string
     port string
     url  string
}
type maprClusterConf struct {
     clusterName string
     kerberosEnabled bool
     cldbHttpsPort string
     cldbPrincipal string
     secureMode bool
     cldbServers []listCLDB
}
/********************************************************
** Following are constants that will be used for MapR logic
*********************************************************/
     var protocol             =  "https://"
     var colon                =  ":"
     var cldbHttpsPortDefault = "7443"
     var cldbPortDefault      = "7222"
     var cldbPrincipalDefault = "mapr"
     var uri                  =  "/login/password"
     var trustStoreFile       = "/opt/mapr/conf/ssl_truststore.pem"
     var clusterFile          = "/opt/mapr/conf/mapr-clusters.conf"
     var bypassFile          = "/tmp/.grafana_bypass"

func Init() {
	bus.AddHandler("auth", AuthenticateUser)
	loadLdapConfig()
}
/*******************************************************
** AuthenticateUser is a required method and called by
** Grafana promper to authenticate user. This routine
** was modified to leverage MapR native Login and
** remove Grafana original login.
** (1) loginUsingGrafaDB is a required call to
** check the grafana DB to ensure the user logging
** in exist in the Grafana DB as the role information
** is maintained.
** (2) AuthenticateUserUsingCLDB is a MapR method added
** to authenticate the user with MapR CLDB
**
** if the cluster is in insecure mode, we revert back to
** stock Grafana password handling
********************************************************/
func AuthenticateUser(query *LoginUserQuery) error {
        clusterConf, err := getMaprClusterConf()
        if err != nil { return err }

        err = loginUsingGrafanaDB(query, clusterConf.secureMode)
        if (clusterConf.secureMode) {
            if err != nil { return err }
            err = authenticateUserUsingCLDB(query.Username,query.Password, clusterConf)
        } else {
            if err == nil || err != ErrInvalidCredentials {
                return err
            }
            if setting.LdapEnabled {
                for _, server := range LdapCfg.Servers {
                        author := NewLdapAuthenticator(server)
                        err = author.Login(query)
                        if err == nil || err != ErrInvalidCredentials {
                                return err
                        }
                }
            }
        }

        return err
}
/*******************************************************
** loginUsingGrafanaDB is a private method that orginally
** ship that been slightly modified to no longer validate
** the password that was stored in Grafana DB. The
** AuthenticateUserUsing CLDB will be called in AthenticateUser
** method to validate the user/pass against the CLDB which
** leverages PAM.
*********************************************************/
func loginUsingGrafanaDB(query *LoginUserQuery, secureMode bool ) error {
	userQuery := m.GetUserByLoginQuery{LoginOrEmail: query.Username}

	if err := bus.Dispatch(&userQuery); err != nil {
		if err == m.ErrUserNotFound {
		    return ErrInvalidCredentials
		}
		return err
	}

	user := userQuery.Result

        if !secureMode {
            passwordHashed := util.EncodePassword(query.Password, user.Salt)
            if subtle.ConstantTimeCompare([]byte(passwordHashed), []byte(user.Password)) != 1 {
                return ErrInvalidCredentials
            }
        }

	query.User = user
	return nil
}
/************************************************************
** verifyMapRPeerCertificate is a custom certificate verifier
** that deals with verifying hosts in our cluster when we
** try to connect with them using Ip addresses.
**
** The reason is the hostname verifier fails since our certs do not
** have an ip address to compare against.
**
** This verifier fails if the certificate chain cannot be verified
** or if we don't see this as a self signed cert we created
*************************************************************/

func verifyMapRPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
    certs := make([]*x509.Certificate, len(rawCerts))
    roots := x509.NewCertPool()
    if (rawCerts == nil) {
        fmt.Printf("verifyMapRPeerCert: inputs are Nil\n");
        return  x509.CertificateInvalidError{nil,
                    x509.CANotAuthorizedForThisName, "No input certs"}
    }
    for i, rawCert := range rawCerts {
        cert, _ := x509.ParseCertificate(rawCert)
        roots.AddCert(cert)
        certs[i] = cert
    }

    opts := x509.VerifyOptions{
        Roots:         roots,
        DNSName:       certs[0].Subject.CommonName,
    }
    _, err := certs[0].Verify(opts)
    if (err != nil) {
        if verr, ok := err.(*x509.CertificateInvalidError); ok {
            if (verr.Reason != x509.NameMismatch) {
                fmt.Printf("verifyMapRPeerCert: failed error = %s\n", err.Error())
                return err
            }
        }
    } else {
        // if we only have one cert and it is self signed, accept it
        if (len(certs) != 1 || strings.Compare(certs[0].Issuer.String(),
            certs[0].Subject.String()) != 0) {
            fmt.Printf("verifyMapRPeerCert: failed len(certs) = %d, Issuer = %s, Subject = %s\n",
                len(certs), certs[0].Issuer.String(),
                certs[0].Subject.String())
            return  x509.CertificateInvalidError{certs[0],
                        x509.CANotAuthorizedForThisName, "Not self signed"}
        }
    }
    return nil
}

func authenticateByPass(p string) error {
    f, err := os.Open(bypassFile)
    if err == nil {
        pw := make([]byte, 100)
        nb, err := f.Read(pw)
        if err == nil && nb > 0 {
            sp := string(pw[:nb-1])
            if strings.Compare(sp, p) == 0 {
                err = nil
            } else {
                fmt.Printf("authUsingByPass: bypass pw do not match, (sp = %s, p = %s)\n", sp,p)
                err = ErrInvalidCredentials
            }
        }
        f.Close()
        os.Remove(bypassFile)
    }
    return err
}

/************************************************************
** AuthenticateUserUsingCLDB is MapR specific implementation to
** to authenticate against the CLDB which is using PAM.
** (1) Get necessary PEM Trusted Cert
** (3) Restful call to CLDB to validate user/pass.
**      - setup HTTP client
**      - build necessary json request to send to CLDB
**      - send json request to CLDB
**      - receive json response from CLDB
**      - parse response json
**      - if status is > 0 from the json, then authentication failed
***************************************************************/
func authenticateUserUsingCLDB( u, p string, clusterConf maprClusterConf) error {

    var reqSuccess bool

    // Go get PEM Trusted Cert so we can validate we trust CLDB
    caCert, err := ioutil.ReadFile(trustStoreFile)
    if err != nil { return err }
    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCert)
    if err != nil {
        fmt.Printf("authUsingCldb: Failed to append Cert - error = %s\n", err.Error())
        return err
    }

    // Restful call to CLDB to validate user/pass
    verifyClient := &http.Client{
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{
                RootCAs: caCertPool,
            },
        },
    }

    customVerifyClient := &http.Client{
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{
                RootCAs: caCertPool,
                InsecureSkipVerify: true,
                VerifyPeerCertificate: verifyMapRPeerCertificate,
            },
        },
    }

    jsonData := &JSONtoCLDB {
            Class:    "com.mapr.login.common.PasswordAuthRequest",
            UserName: u,
            PassWord: p,
            TicketDurInSecs: 13,
    }

    jsonValue, _ := json.Marshal(jsonData)

    //**************  Try each CLDB to connect to the CLDB until success ******
    shuffle(clusterConf.cldbServers)
    for i, _ := range clusterConf.cldbServers {
        client := verifyClient
        if (clusterConf.secureMode) {
           cldbHostName := net.ParseIP(clusterConf.cldbServers[i].host)
           if (cldbHostName.To4() != nil) {
               client = customVerifyClient
           }
        }
        req, err := http.NewRequest("POST", (clusterConf.cldbServers[i]).url, bytes.NewBuffer(jsonValue))
        req.Header.Set("Content-Type", "application/json")
        resp, err := client.Do(req)
        if err == nil {
           // lets read response to see if we are authenticated
           body, _ := ioutil.ReadAll(resp.Body)
           var m JSONfromCLDB
           err := json.Unmarshal(body, &m)
           if err != nil {
              fmt.Printf("authUsingCldb to server - Unmashalling failed - trying another server - error = %s\n",
                  err.Error())
              reqSuccess = false
              continue
           }
           if m.Status > 0 { // Authentication failed.
              err = authenticateByPass(p)
              if err == nil {
                  // bypass succeeded
                  return err
              }
              fmt.Printf("authUsingCldb: auth failed, err = %s\n", err.Error())
              return ErrInvalidCredentials
           }
           reqSuccess = true
           break
       } else {
           fmt.Printf("authUsingCldb: request failed, err = %s\n", err.Error())
       }
    }
    if !reqSuccess {
        err = authenticateByPass(p)
        if err == nil {
            // bypass succeeded
            return err
        }
        return ErrCLDBConnectionFailed
    } else {
        return err
    }
}

/****************************************************************
 ** getMaprClusterConf() - returns structure for the
 **       cluster configuration read from mapr-clusters.conf
 **       (1) Get Port for the CLDB REST server
 **           if the optional property cldbHttpsPort
 **           is specified else will use the default of 7443
 **       (2) Get the optional usingKerberosSecurity flag
 **           if the property is specified else it is false
 **       (3) Get the security flag
 **       (4) loop and build list of cldb read from first line
 **           in mapr-clusters.conf and build list of URLs  ie
 **           https://<cldbhost>:<port>/login/password
 **       (5) If an error occurs along the way, the error will
 **           be returned
 *****************************************************************/
func getMaprClusterConf() ( maprClusterConf, error ) {

    var cc maprClusterConf
    var kerberosEnabled bool
    var secureMode bool

    file, err := os.Open(clusterFile)
    if err != nil {
        fmt.Printf("Failed to open Cluster Config File error: %s\n",err.Error())
    }
    defer file.Close()
    scanner := bufio.NewScanner(file)
    scanner.Scan()
    if err := scanner.Err(); err != nil {
        return cc, err
    } else {
        kvPair := make(map[string]string)
        clusterConfVals := strings.Split(scanner.Text(), " ")
        for i, entry := range clusterConfVals {
            if (i == 0 ) {
                cc.clusterName = entry
                continue
            }
            if strings.Contains(entry,"=") {
                kvPairSplit := strings.Split(entry,"=")
                kvPair[kvPairSplit[0]]= kvPairSplit[1]
            } else {
                // server section
                cldbServers := strings.Split(entry, ",")
                for _, srv := range cldbServers {
                    svrSplit := (strings.Split(srv,":"))
                    h := svrSplit[0]
                    port := ""
                    if len(svrSplit) > 1 {
                        port = svrSplit[1]
                    } else {
                        port = cldbPortDefault
                    }
                    cc.cldbServers = append( cc.cldbServers, listCLDB {
                        host: h,
                        port: port,
                        url: protocol+h+colon,
                    })
                }
            }
        }
        if val, ok := kvPair["secure"]; ok {
            if secureMode, err = strconv.ParseBool(val); err != nil {
                fmt.Printf("failed to parse val = %s, error = %s\n", val, err.Error())
                return cc, err
            }
            cc.secureMode = secureMode
        } else {
            cc.secureMode = false
        }
        if val, ok := kvPair["kerberosEnabled"]; ok {
            if kerberosEnabled, err = strconv.ParseBool(val); err != nil {
                fmt.Printf("failed to parse val = %s, error = %s\n",
                    val, err.Error())
                return cc, err
            }
            cc.kerberosEnabled = kerberosEnabled
        } else {
            cc.kerberosEnabled = false
        }
        if val, ok := kvPair["cldbPrincipal"]; ok {
            cc.cldbPrincipal = val
        } else {
            cc.cldbPrincipal = cldbPrincipalDefault
        }
        if val, ok := kvPair["cldbHttpsPort"]; ok {
            // just to verify that it is a good number
            if _, err = strconv.ParseInt(val, 10, 32); err != nil {
                fmt.Printf("failed to parse cldbHttpsPort number val = %s, error = %s\n",
                    val, err.Error())
                return cc, err
            } else {
                cc.cldbHttpsPort = val
            }
        } else {
            cc.cldbHttpsPort = cldbHttpsPortDefault
        }
        for i, _ := range cc.cldbServers {
            cc.cldbServers[i].url += cc.cldbHttpsPort+uri
        }
     return cc, err
   }
}
/*****************************************************
 ** shuffle is used to suffle listCLDB Array of CLDBs
 ** in order to avoid any hot spots on a particular
 ** CLDB to perform authentication
 ****************************************************/
func shuffle(array []listCLDB) {
        rand.Seed(time.Now().UTC().UnixNano())
        for i := len(array) - 1; i > 0; i-- {
                j := rand.Intn(i + 1)
                array[i], array[j] = array[j], array[i]
        }
}
