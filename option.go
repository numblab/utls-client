package client

type option struct {
	hosts map[string]string
}

type OptionFn func(*option)

func WithHost(host map[string]string) OptionFn {
	return func(o *option) {
		o.hosts = host
	}
}
