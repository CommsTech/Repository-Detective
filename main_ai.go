package main

import (
	"fmt"

	"git.commsnet.org/commstech/repository-detective/ai"
	"github.com/gin-gonic/gin"
)

type aiStatusBridge struct{}

func (aiStatusBridge) Status(c *gin.Context) ai.ProviderStatus {
	st := ai.GetProviderStatus()
	if aiClient == nil {
		st.Configured = false
		if !config.needsAIProvider() {
			st.PolicyDisabled = true
		}
		return st
	}
	st.Configured = true
	st.Provider = string(aiClient.Provider())
	st.Model = aiClient.Model()
	st.TestMode = ai.ConnectionTestMode(config.AIConnectionTestMode)
	return st
}

func (aiStatusBridge) TestConnection(c *gin.Context, force bool) (ai.ProviderStatus, error) {
	if aiClient == nil {
		return ai.ProviderStatus{}, fmt.Errorf("ai client not configured")
	}
	mode := ai.ConnectionTestMode(config.AIConnectionTestMode)
	if mode == "" {
		mode = ai.TestModeMetadataOnly
	}
	if mode == ai.TestModeChatCompletion {
		logger.Warn("Manual AI connection test using chat completion — may incur API cost")
	}
	return ai.RunConnectionTest(c.Request.Context(), aiClient, mode, force)
}

func initAIStatus() {
	ai.ConfigureStatusCache(config.AIConnectionTestCacheMinutes)
	st := ai.ProviderStatus{
		TestMode: ai.ConnectionTestMode(config.AIConnectionTestMode),
	}
	if config.AIConnectionTestMode == "" {
		st.TestMode = ai.TestModeMetadataOnly
	}
	if !config.needsAIProvider() {
		st.PolicyDisabled = true
	}
	ai.SetInitialProviderStatus(st)
}
