package config

import (
	"fmt"
	"strconv"
	"time"
)

const (
	envProxyEnable       = "GIT_GEN_PROXY_ENABLE"
	envProxyType         = "GIT_GEN_PROXY_TYPE"
	envProxyAddr         = "GIT_GEN_PROXY_ADDR"
	envProxyLogin        = "GIT_GEN_PROXY_LOGIN"
	envProxyPassw        = "GIT_GEN_PROXY_PASSW"
	envRetryRetries      = "GIT_GEN_LLM_RETRY_RETRIES"
	envRetryInitInterval = "GIT_GEN_LLM_RETRY_INIT_INTERVAL"
	envRetryMaxInterval  = "GIT_GEN_LLM_RETRY_MAX_INTERVAL"
	envOpenAIToken       = "GIT_GEN_OPENAI_TOKEN"
	envOpenAIModel       = "GIT_GEN_OPENAI_MODEL"
)

func (llms LLMs) ApplyDotEnv(values map[string]string) error {
	for index := range llms {
		if err := llms[index].applyDotEnv(values); err != nil {
			return fmt.Errorf("llm %q: %w", llms[index].Name, err)
		}
	}

	return nil
}

func (llm *LLM) applyDotEnv(values map[string]string) error {
	if err := llm.Proxy.applyDotEnv(values); err != nil {
		return err
	}
	if err := llm.Retry.applyDotEnv(values); err != nil {
		return err
	}

	llm.OpenAI.applyDotEnv(values)

	return nil
}

func (proxy *Proxy) applyDotEnv(values map[string]string) error {
	if value, ok := values[envProxyEnable]; ok {
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse %s: %w", envProxyEnable, err)
		}

		proxy.Enable = enabled
	}

	if value, ok := values[envProxyType]; ok {
		proxy.Typ = TypeProxy(value)
	}
	if value, ok := values[envProxyAddr]; ok {
		proxy.Addr = value
	}
	if value, ok := values[envProxyLogin]; ok {
		proxy.Login = value
	}
	if value, ok := values[envProxyPassw]; ok {
		proxy.Passw = value
	}

	return nil
}

func (retry *Retry) applyDotEnv(values map[string]string) error {
	if err := retry.applyRetries(values); err != nil {
		return err
	}

	return retry.applyIntervals(values)
}

func (retry *Retry) applyRetries(values map[string]string) error {
	value, ok := values[envRetryRetries]
	if !ok {
		return nil
	}

	retries, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return fmt.Errorf("parse %s: %w", envRetryRetries, err)
	}

	retry.Retries = retries

	return nil
}

func (retry *Retry) applyIntervals(values map[string]string) error {
	if value, ok := values[envRetryInitInterval]; ok {
		interval, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("parse %s: %w", envRetryInitInterval, err)
		}

		retry.InitInterval = interval
	}

	if value, ok := values[envRetryMaxInterval]; ok {
		interval, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("parse %s: %w", envRetryMaxInterval, err)
		}

		retry.MaxInterval = interval
	}

	return nil
}

func (openAI *OpenAI) applyDotEnv(values map[string]string) {
	if value, ok := values[envOpenAIToken]; ok {
		openAI.Token = value
	}
	if value, ok := values[envOpenAIModel]; ok {
		openAI.Model = value
	}
}
