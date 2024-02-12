package server

type Config struct {
	network             string
	symbol              string
	httpPort            int
	interval            int
	payout              int
	proxyCount          int
	hcaptchaSiteKey     string
	hcaptchaSecret      string
	discordClientId     string
	discordClientSecret string
	discordRedirectUrl  string
}

func NewConfig(network, symbol string, httpPort, interval, payout, proxyCount int, hcaptchaSiteKey, hcaptchaSecret, discordClientId, discordClientSecret, discordRedirectUrl string) *Config {
	return &Config{
		network:             network,
		symbol:              symbol,
		httpPort:            httpPort,
		interval:            interval,
		payout:              payout,
		proxyCount:          proxyCount,
		hcaptchaSiteKey:     hcaptchaSiteKey,
		hcaptchaSecret:      hcaptchaSecret,
		discordClientId:     discordClientId,
		discordClientSecret: discordClientSecret,
		discordRedirectUrl:  discordRedirectUrl,
	}
}
