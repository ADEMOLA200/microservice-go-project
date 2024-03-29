package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/showbaba/microservice-sample-go/shared"
	"gopkg.in/yaml.v2"
)

func ParseRequest(r *http.Request) *RequestInfo {
	return &RequestInfo{
		IPAddress:   r.RemoteAddr,
		Token:       r.Header.Get("token"),
		Host:        r.Host,
		UserAgent:   r.Header.Get("User-Agent"),
		Method:      r.Method,
		OriginalURL: r.URL.RequestURI(),
		Query:       r.URL.RawQuery,
		Params:      []string{r.URL.Path},
		Body:        r.Body,
		Auth:        r.Header.Get("Authorization"),
	}
}

type RequestInfo struct {
	IPAddress   string
	Token       string
	Host        string
	UserAgent   string
	Method      string
	OriginalURL string
	Query       string
	Params      []string
	Auth        string
	Body        io.Reader
}

func ReadConfig() (*ServiceConfig, error) {
	data, err := ioutil.ReadFile("conf.yaml")
	if err != nil {
		return nil, err
	}
	var config ServiceConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

type Request struct {
	Method      string
	URL         string
	Auth        string
	GatewayAuth string
}

type Response struct {
	Status     string
	StatusCode int
	Data       any
}

func ForwardRequest(r *Request, data *[]byte) (*shared.APIResponse, error) {
	var (
		req  *http.Request
		body *bytes.Buffer
		err  error
	)
	if data != nil {
		body = bytes.NewBuffer(*data)
		req, err = http.NewRequest(r.Method, r.URL, body)
		if err != nil {
			return nil, err
		}
	} else {
		req, err = http.NewRequest(r.Method, r.URL, nil)
		if err != nil {
			return nil, err
		}
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("gateway_signature", r.GatewayAuth)
	req.Header.Add("Authorization", fmt.Sprintf(`Bearer %s`, r.Auth))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	var respData shared.APIResponse
	err = json.Unmarshal(b, &respData)
	if err != nil {
		return nil, err
	}
	return &respData, nil
}

func GenerateGatewayToken(service string) (string, error) {
	claims := &shared.GatewayTokenJwtClaim{
		Gateway: shared.GATEWAY_SERVICE,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Local().Add(time.Hour * time.Duration(24)).Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	var secretKey string
	switch service {
	case shared.AUTH_SERVICE:
		secretKey = GetConfig().AuthServiceJWTSecretKey
	case shared.USER_SERVICE:
		secretKey = GetConfig().UserServiceJWTSecretKey
	case shared.POST_SERVICE:
		secretKey = GetConfig().PostServiceSecretKey
	default:
		secretKey = ""
	}
	signedToken, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

type ServiceConfig struct {
	Services struct {
		Auth struct {
			Port      int    `yaml:"port"`
			BaseURL   string `yaml:"base_url"`
			Endpoints struct {
				Post []string `yaml:"post"`
				Get  []string `yaml:"get"`
			} `yaml:"endpoints"`
		} `yaml:"auth"`
		User struct {
			Port      int    `yaml:"port"`
			BaseURL   string `yaml:"base_url"`
			Endpoints struct {
				Post []string `yaml:"post"`
				Get  []string `yaml:"get"`
			} `yaml:"endpoints"`
		} `yaml:"user"`
		Post struct {
			Port      int    `yaml:"port"`
			BaseURL   string `yaml:"base_url"`
			Endpoints struct {
				Post []string `yaml:"post"`
				Get  []string `yaml:"get"`
			} `yaml:"endpoints"`
		} `yaml:"post"`
	} `yaml:"services"`
}
