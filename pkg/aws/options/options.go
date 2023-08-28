package options

type Options struct {
	AWSRegion  string
	AWSRoleArn string
}

type Option interface {
	Apply(*Options)
}

type awsRegion string

func (o awsRegion) Apply(i *Options) {
	if o != "" {
		i.AWSRegion = string(o)
	}
}

func WithRegion(d string) Option {
	return awsRegion(d)
}

type awsRoleArn string

func (o awsRoleArn) Apply(i *Options) {
	if o != "" {
		i.AWSRoleArn = string(o)
	}
}

func WithRoleArn(d string) Option {
	return awsRoleArn(d)
}
