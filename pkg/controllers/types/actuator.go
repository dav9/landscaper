// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import (
	"context"
	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

func NewActuator() (reconcile.Reconciler, error) {
	return &actuator{}, nil
}

type actuator struct {
	log logr.Logger
	c   client.Client
}

var _ inject.Client = &actuator{}

var _ inject.Logger = &actuator{}

// InjectClients injects the current kubernetes client into the actuator
func (a *actuator) InjectClient(c client.Client) error {
	a.c = c
	return nil
}

// InjectLogger injects a logging instance into the actuator
func (a *actuator) InjectLogger(log logr.Logger) error {
	a.log = log
	return nil
}

func (a *actuator) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	defer ctx.Done()
	a.log.Info("reconcile", "resource", req.NamespacedName)

	customType := &v1alpha1.Type{}
	if err := a.c.Get(ctx, req.NamespacedName, customType); err != nil {
		a.log.Error(err, "unable to get resource")
		return reconcile.Result{}, err
	}

	if customType.Status.ObservedGeneration == customType.Generation {
		return reconcile.Result{}, nil
	}

	customType.Status.ObservedGeneration = customType.Generation

	customType.Status.Conditions = v1alpha1.CreateOrUpdateConditions(customType.Status.Conditions, v1alpha1.TypeEstablished,
		v1alpha1.ConditionTrue, "", "")

	if err := a.c.Status().Update(ctx, customType); err != nil {
		a.log.Error(err, "unable to update status")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}