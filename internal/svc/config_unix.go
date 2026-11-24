//go:build !windows

package svc

import "github.com/kardianos/service"

func platformOptions() service.KeyValue {
	return service.KeyValue{
		"UserService": true,
	}
}
