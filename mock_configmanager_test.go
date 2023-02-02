// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package core

import (
	"github.com/couchbase/gocbcorex/contrib/cbconfig"
	"sync"
)

// Ensure, that ConfigManagerMock does implement ConfigManager.
// If this is not the case, regenerate this file with moq.
var _ ConfigManager = &ConfigManagerMock{}

// ConfigManagerMock is a mock implementation of ConfigManager.
//
//	func TestSomethingThatUsesConfigManager(t *testing.T) {
//
//		// make and configure a mocked ConfigManager
//		mockedConfigManager := &ConfigManagerMock{
//			ApplyConfigFunc: func(sourceHostname string, json *cbconfig.TerseConfigJson)  {
//				panic("mock out the ApplyConfig method")
//			},
//			RegisterCallbackFunc: func(handler RouteConfigHandler)  {
//				panic("mock out the RegisterCallback method")
//			},
//			UnregisterCallbackFunc: func(handler RouteConfigHandler)  {
//				panic("mock out the UnregisterCallback method")
//			},
//		}
//
//		// use mockedConfigManager in code that requires ConfigManager
//		// and then make assertions.
//
//	}
type ConfigManagerMock struct {
	// ApplyConfigFunc mocks the ApplyConfig method.
	ApplyConfigFunc func(sourceHostname string, json *cbconfig.TerseConfigJson)

	// RegisterCallbackFunc mocks the RegisterCallback method.
	RegisterCallbackFunc func(handler RouteConfigHandler)

	// UnregisterCallbackFunc mocks the UnregisterCallback method.
	UnregisterCallbackFunc func(handler RouteConfigHandler)

	// calls tracks calls to the methods.
	calls struct {
		// ApplyConfig holds details about calls to the ApplyConfig method.
		ApplyConfig []struct {
			// SourceHostname is the sourceHostname argument value.
			SourceHostname string
			// JSON is the json argument value.
			JSON *cbconfig.TerseConfigJson
		}
		// RegisterCallback holds details about calls to the RegisterCallback method.
		RegisterCallback []struct {
			// Handler is the handler argument value.
			Handler RouteConfigHandler
		}
		// UnregisterCallback holds details about calls to the UnregisterCallback method.
		UnregisterCallback []struct {
			// Handler is the handler argument value.
			Handler RouteConfigHandler
		}
	}
	lockApplyConfig        sync.RWMutex
	lockRegisterCallback   sync.RWMutex
	lockUnregisterCallback sync.RWMutex
}

// ApplyConfig calls ApplyConfigFunc.
func (mock *ConfigManagerMock) ApplyConfig(sourceHostname string, json *cbconfig.TerseConfigJson) {
	if mock.ApplyConfigFunc == nil {
		panic("ConfigManagerMock.ApplyConfigFunc: method is nil but ConfigManager.ApplyConfig was just called")
	}
	callInfo := struct {
		SourceHostname string
		JSON           *cbconfig.TerseConfigJson
	}{
		SourceHostname: sourceHostname,
		JSON:           json,
	}
	mock.lockApplyConfig.Lock()
	mock.calls.ApplyConfig = append(mock.calls.ApplyConfig, callInfo)
	mock.lockApplyConfig.Unlock()
	mock.ApplyConfigFunc(sourceHostname, json)
}

// ApplyConfigCalls gets all the calls that were made to ApplyConfig.
// Check the length with:
//
//	len(mockedConfigManager.ApplyConfigCalls())
func (mock *ConfigManagerMock) ApplyConfigCalls() []struct {
	SourceHostname string
	JSON           *cbconfig.TerseConfigJson
} {
	var calls []struct {
		SourceHostname string
		JSON           *cbconfig.TerseConfigJson
	}
	mock.lockApplyConfig.RLock()
	calls = mock.calls.ApplyConfig
	mock.lockApplyConfig.RUnlock()
	return calls
}

// RegisterCallback calls RegisterCallbackFunc.
func (mock *ConfigManagerMock) RegisterCallback(handler RouteConfigHandler) {
	if mock.RegisterCallbackFunc == nil {
		panic("ConfigManagerMock.RegisterCallbackFunc: method is nil but ConfigManager.RegisterCallback was just called")
	}
	callInfo := struct {
		Handler RouteConfigHandler
	}{
		Handler: handler,
	}
	mock.lockRegisterCallback.Lock()
	mock.calls.RegisterCallback = append(mock.calls.RegisterCallback, callInfo)
	mock.lockRegisterCallback.Unlock()
	mock.RegisterCallbackFunc(handler)
}

// RegisterCallbackCalls gets all the calls that were made to RegisterCallback.
// Check the length with:
//
//	len(mockedConfigManager.RegisterCallbackCalls())
func (mock *ConfigManagerMock) RegisterCallbackCalls() []struct {
	Handler RouteConfigHandler
} {
	var calls []struct {
		Handler RouteConfigHandler
	}
	mock.lockRegisterCallback.RLock()
	calls = mock.calls.RegisterCallback
	mock.lockRegisterCallback.RUnlock()
	return calls
}

// UnregisterCallback calls UnregisterCallbackFunc.
func (mock *ConfigManagerMock) UnregisterCallback(handler RouteConfigHandler) {
	if mock.UnregisterCallbackFunc == nil {
		panic("ConfigManagerMock.UnregisterCallbackFunc: method is nil but ConfigManager.UnregisterCallback was just called")
	}
	callInfo := struct {
		Handler RouteConfigHandler
	}{
		Handler: handler,
	}
	mock.lockUnregisterCallback.Lock()
	mock.calls.UnregisterCallback = append(mock.calls.UnregisterCallback, callInfo)
	mock.lockUnregisterCallback.Unlock()
	mock.UnregisterCallbackFunc(handler)
}

// UnregisterCallbackCalls gets all the calls that were made to UnregisterCallback.
// Check the length with:
//
//	len(mockedConfigManager.UnregisterCallbackCalls())
func (mock *ConfigManagerMock) UnregisterCallbackCalls() []struct {
	Handler RouteConfigHandler
} {
	var calls []struct {
		Handler RouteConfigHandler
	}
	mock.lockUnregisterCallback.RLock()
	calls = mock.calls.UnregisterCallback
	mock.lockUnregisterCallback.RUnlock()
	return calls
}
