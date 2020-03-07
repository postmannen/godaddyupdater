package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// getPublicIP will get the public ip from ipify.org
func (s *server) getPublicIP() (string, error) {
	url := "https://api.ipify.org?format=text" // we are using a pulib IP API, we're using ipify here, below are some others
	// https://www.ipify.org
	// http://myexternalip.com
	// http://api.ident.me
	// http://whatismyipaddress.com/api
	var ip []byte

	for {
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("failed getting public ip %v\n", err)
			log.Println("Sleeping for some seconds before retrying......")
			s.internetOK.Set(0)
			time.Sleep(time.Second * 30)
			continue
		}
		defer resp.Body.Close()
		ip, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("failed reading public ip body %v", err)
			log.Println("Sleeping for some seconds before retrying......")
			s.internetOK.Set(0)
			time.Sleep(time.Second * 30)
			continue
		}

		// If all above was ok we break out...
		break
	}

	s.internetOK.Set(1)
	return string(ip), nil
}

// getgodaddyCurrentIP will retrieve the currently registered ip address for a domain at godaddy.
func getGodaddyCurrentIP(key string, secret string, domain string, subDomain string) (string, error) {
	httpClient := &http.Client{}
	// Create a get request
	req, err := http.NewRequest("GET", "https://api.godaddy.com/v1/domains/"+domain+"/records/A/"+subDomain, nil)
	if err != nil {
		return "", fmt.Errorf("failed creating request: %v", err)
	}

	// Set the correct header for the request.
	req.Header.Set("Authorization", "sso-key "+key+":"+secret)
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed getting response: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("wrong status code %v", resp.Status)
	}

	// If the get request went ok, get the body of the response.
	var body []byte
	if resp.StatusCode == http.StatusOK {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed reading body : %v", err)
		}

		body = b
	}

	// Godaddy expects the body to be of type JSON array, so we make it into a slice,
	// so it get's marshaled correctly.
	GData := []goDaddyData{}
	json.Unmarshal(body, &GData)

	return GData[0].Data, err

}

func setGodaddyCurrentIP(key string, secret string, apiURL string, gdData string) error {
	// Prepare the http client
	httpClient := &http.Client{}
	// Create a new POST request, and prepare it with the POST data.
	req, err := http.NewRequest("PUT", apiURL, strings.NewReader(gdData))
	if err != nil {
		return fmt.Errorf("failed creating request: %v", err)
	}

	req.Header.Set("Authorization", "sso-key "+key+":"+secret)
	req.Header.Set("Content-Type", "application/json")

	// Execute the http request, and set new ip.
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed doing POST: %v", err)
	}

	defer resp.Body.Close()

	// Read the response, and check if all went OK.
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error: failed reading the response body of the POST: ", err)
	}

	// Empty string response indicates OK.
	if string(b) == "" {
		log.Println("Updating godaddy DNS record, OK")
	} else {
		log.Println("Warning: godaddy update problem: ", string(b))
	}

	return nil

}

// goDaddyData reflects how godaddy wants the data in JSON format.
type goDaddyData struct {
	Data string `json:"data"`
	TTL  int    `json:"ttl"`
}

// run will orchestrate the checks for finding out if ip's are changed,
// and change it at godaddy if changed.
func run(key string, secret string, checkInterval int, domain string, subDomain string, s *server) error {
	// Convert the check interval from int to Duration.
	interval := time.Duration(checkInterval) * time.Second
	pIPCh := make(chan string)

	// get current ip registered at godaddy.
	gIP, err := getGodaddyCurrentIP(key, secret, domain, subDomain)
	if err != nil {
		log.Printf("error: failed to get ip from godaddy %v\n", err)
	}

	if gIP == "" && err != nil {
		return fmt.Errorf("the ip value returned from godaddy returned blank, credential issue ?")
	}
	log.Printf("Current godaddy ip for "+subDomain+"."+domain+" = %v\n", gIP)

	// Continously at the given interval check the current public IP,
	go func() {
		for {
			// Wait the interval given before checking current ip
			<-time.After(interval)

			// Get the current public ip of your connection.
			p, err := s.getPublicIP()
			if err != nil {
				log.Println("error: public ip: ", err)
				return
			}
			pIPCh <- p
		}
	}()

	// Check if changed, and set new value at godaddy if changed.
	for {
		pIP := <-pIPCh
		log.Printf("My current locally detected public IP is:%s\n", pIP)

		// If the current public ip and the registered dns ip at godaddy are not the same,
		// change the value in the godaddy dns record.
		if pIP != gIP {
			log.Println("* The ip's are different, preparing to update record at godaddy.")
			gd := goDaddyData{
				Data: pIP,
				TTL:  600,
			}

			// Create the data for header that will be changed
			gdArray := []goDaddyData{gd}
			gdJSON, err := json.Marshal(gdArray)
			if err != nil {
				log.Println("error: json marshal failed")
			}

			apiURL := "https://api.godaddy.com/v1/domains/" + domain + "/records/A/" + subDomain

			// do the api call to set the new ip
			err = setGodaddyCurrentIP(key, secret, apiURL, string(gdJSON))
			if err != nil {
				log.Println("error: setGodaddyCurrent ip = ", err)
			}

			// We really only need to ask the godaddy API once in the beginning for
			// the public IP registered for a domain. After that we can keep a local
			// record for it, and there will be no need ask goDaddy again.
			gIP = pIP
		}

		log.Println("The ip address have not changed, keeping everything as it is.")

	}
}

type server struct {
	internetOK prometheus.Gauge
}

func newServer() *server {
	s := server{
		internetOK: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "internet_ok",
			Help: "Internet up or down",
		}),
	}
	// set default to ok
	s.internetOK.Set(1)
	return &s
}

// startPrometheus will start the http listener for the
// prometheus exporter.
func (s *server) startPrometheus(port string) {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":"+port, nil)
	}()
}

type flags struct {
	auth          string
	key           string
	secret        string
	checkInterval int
	domain        string
	subDomain     string
	promExpPort   string
}

func newFlags() *flags {
	auth := flag.String("auth", "env",
		`Use "env" or "flag" for way to get key and secret.
		If value chosen is "flag", use the -key and -secret flags.
		If value chosen is "env", set the env variables "goddaddykey" and "godaddysecret".
	`)
	key := flag.String("key", "", "the key you got at https://developer.godaddy.com/keys")
	secret := flag.String("secret", "", "the secret you got at https://developer.godaddy.com/keys")
	checkInterval := flag.Int("checkInterval", 5, "check interval in seconds")
	domain := flag.String("domain", "",
		`domain name, e.g. -domain="erter.org. NB: If you want to update the main domain like erter.org use "@" as value with the subDomain flag like  -subDomain="@""`)
	subDomain := flag.String("subDomain", "",
		`domain name, e.g. -subDomain="dev". NB: If you want to update the main domain like erter.org use "@" as value like -subDomain="@"`)
	promExpPort := flag.String("promExpPort", "2112",
		`The port number to run the prometheus exporter on written as a string value. Default : -promExpPort="2112"`)
	flag.Parse()

	// Dereference all the pointer values and put them in as struct values.
	f := flags{
		auth:          *auth,
		key:           *key,
		secret:        *secret,
		checkInterval: *checkInterval,
		domain:        *domain,
		subDomain:     *subDomain,
		promExpPort:   *promExpPort,
	}

	return &f
}

func checkFlags(f *flags) error {
	switch f.auth {
	case "env":
		log.Println("Info: Using default auth method env")
		f.key = os.Getenv("godaddykey")
		f.secret = os.Getenv("godaddysecret")
		if f.key == "" || f.secret == "" {
			return fmt.Errorf("method env chosen, and you need to set key and secret")
		}
	case "flag":
		if f.key == "" || f.secret == "" {
			return fmt.Errorf("method flag chosen, and you need to set key and secret")
		}
	}

	if f.domain == "" {
		return fmt.Errorf("No domain specified, please specify a domain with the -domain flag")
	}

	if f.subDomain == "" {
		return fmt.Errorf("No sub domain specified, please specify a sub domain with the -subDomain flag")
	}

	return nil
}

func main() {
	// get the flags entered when starting the program
	f := newFlags()

	err := checkFlags(f)
	if err != nil {
		log.Printf("error: %v\n", err)
		return
	}

	s := newServer()
	s.startPrometheus(f.promExpPort)

	// Run the checking, and eventually edit dns record at godaddy.
	run(f.key, f.secret, f.checkInterval, f.domain, f.subDomain, s)

}
