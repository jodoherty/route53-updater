package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type Service struct {
	config Config
	router http.Handler
}

func CreateService(config Config) *Service {
	router := httprouter.New()
	s := Service{
		config: config,
		router: router,
	}
	router.GET("/", s.getIndex)
	router.POST("/update/:hostname/:token", s.postUpdate)
	router.POST("/update/:hostname/:token/:ip", s.postUpdateForIp)
	return &s
}

func isIpAddress(ip string) bool {
	matched, err := regexp.MatchString(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`, ip)
	if err != nil {
		return false
	}
	return matched
}

func (s *Service) getIndex(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, `{
  "id": "route53-updater",
  "api-version": "1.0.0"
}
`)
}

func (s *Service) postUpdate(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	// Pull IP Address from request
	ip := strings.Split(r.RemoteAddr, ":")[0] // default
	// Replace with forwarded header if exists and proxy IP is trusted
	forwarded := r.Header.Get("X-Forwarded-For")
	if s.config.TrustsProxy(ip) && forwarded != "" {
		ips := strings.Split(forwarded, ",")
		validIps := []string{}
		for i := range ips {
			ips[i] = strings.TrimSpace(ips[i])
			if isIpAddress(ips[i]) {
				validIps = append(validIps, ips[i])
			}
		}
		if len(validIps) > 0 {
			ip = validIps[0]
		}
	}
	// Forward to generic handler
	params = append(params, httprouter.Param{"ip", ip})
	s.postUpdateForIp(w, r, params)
}

func (s *Service) postUpdateForIp(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	hostname := params.ByName("hostname")
	token := params.ByName("token")
	ip := params.ByName("ip")
	if !isIpAddress(ip) {
		log.Printf(`Received invalid ip address: "%s"`, ip)
		http.Error(w, "Received invalid ip address", http.StatusInternalServerError)
		return
	}
	if !s.config.IsValidToken(hostname, token) {
		log.Printf(`Received invalid hostname/token: "%s" "%s"`, hostname, token)
		http.Error(w, "Invalid hostname/token pair", http.StatusInternalServerError)
		return
	}

	client := route53.New(session.New(), aws.NewConfig().WithRegion(s.config.Region(hostname)))
	_, err := client.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("UPSERT"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(hostname),
						Type: aws.String("A"),
						TTL:  aws.Int64(300),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(ip),
							},
						},
					},
				},
			},
			Comment: aws.String("route53-updater update"),
		},
		HostedZoneId: aws.String(s.config.ZoneId(hostname)),
	})
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf(`Updated "%s" to "%s"`, hostname, ip)
	fmt.Fprint(w, "SUCCESS")
}

func (s *Service) Start(addr string) {
	log.Printf(`Starting server on "%s"`, addr)
	log.Fatal(http.ListenAndServe(addr, s.router))
}
