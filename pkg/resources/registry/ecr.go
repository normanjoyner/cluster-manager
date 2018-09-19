package registry

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"

	csv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
)

// ECR defines a registry on AWS
type ECR struct {
	Credentials map[string]string
}

// AccessKey returns the access key id from credentials
func (e ECR) AccessKey() string {
	return e.Credentials["aws_access_key_id"]
}

// SecretKey returns the secret key from credentials
func (e ECR) SecretKey() string {
	return e.Credentials["aws_secret_access_key"]
}

// Region returns the region to auth to on AWS, with a default return value of
// 'us-east-1'
func (e ECR) Region() string {
	if e.Credentials["region"] != "" {
		return e.Credentials["region"]
	}

	return "us-east-1"
}

func (e ECR) newEcrClient() *ecr.ECR {
	sess := session.Must(session.NewSession())
	creds := credentials.NewStaticCredentials(e.AccessKey(), e.SecretKey(), "")
	awsConfig := aws.NewConfig().WithRegion(e.Region()).WithCredentials(creds)

	return ecr.New(sess, awsConfig)
}

// CreateAuthToken returns a token that can be used to authenticate
// with AWS registries
func (e ECR) CreateAuthToken() (csv3.AuthTokenDef, error) {
	c := e.newEcrClient()
	params := &ecr.GetAuthorizationTokenInput{}

	resp, err := c.GetAuthorizationToken(params)
	if err != nil {
		return csv3.AuthTokenDef{}, err
	}

	token := resp.AuthorizationData[0]
	expires := *token.ExpiresAt
	return csv3.AuthTokenDef{
		Token:    *token.AuthorizationToken,
		Endpoint: *token.ProxyEndpoint,
		Type:     DockerJSON,
		Expires:  expires.String(),
	}, nil
}
