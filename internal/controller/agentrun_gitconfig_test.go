/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"

	konveyoriov1alpha1 "github.com/konveyor/agentic-controller/api/v1alpha1"
)

// Shared git identity fixtures: an Agent-level default and a run-level
// override, reused across the resolution, env-emission, and CRD-validation
// tests.
const (
	gitNameAgent  = "Coolstore Bot"
	gitEmailAgent = "bot@myorg.com"
	gitNameRun    = "Jane Dev"
	gitEmailRun   = "jane@myorg.com"
)

func gitCfg(name, email string) *konveyoriov1alpha1.GitConfig {
	return &konveyoriov1alpha1.GitConfig{UserName: name, UserEmail: email}
}

func TestResolveGitIdentity(t *testing.T) {
	tests := []struct {
		name      string
		agent     *konveyoriov1alpha1.GitConfig
		run       *konveyoriov1alpha1.GitConfig
		wantName  string
		wantEmail string
	}{
		{name: "both unset"},
		{
			name:      "agent only",
			agent:     gitCfg(gitNameAgent, gitEmailAgent),
			wantName:  gitNameAgent,
			wantEmail: gitEmailAgent,
		},
		{
			name:      "run only",
			run:       gitCfg(gitNameRun, gitEmailRun),
			wantName:  gitNameRun,
			wantEmail: gitEmailRun,
		},
		{
			// The run's identity replaces the agent's wholesale; a
			// both-or-neither CRD constraint rules out partial overrides.
			name:      "run replaces agent",
			agent:     gitCfg(gitNameAgent, gitEmailAgent),
			run:       gitCfg(gitNameRun, gitEmailRun),
			wantName:  gitNameRun,
			wantEmail: gitEmailRun,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &konveyoriov1alpha1.Agent{}
			agent.Spec.GitConfig = tt.agent
			run := &konveyoriov1alpha1.AgentRun{}
			run.Spec.GitConfig = tt.run

			gotName, gotEmail := resolveGitIdentity(agent, run)
			if gotName != tt.wantName {
				t.Errorf("name = %q, want %q", gotName, tt.wantName)
			}
			if gotEmail != tt.wantEmail {
				t.Errorf("email = %q, want %q", gotEmail, tt.wantEmail)
			}
		})
	}
}

func TestBuildEnvVarsGitIdentity(t *testing.T) {
	r := &AgentRunReconciler{}

	findEnv := func(env []corev1.EnvVar, name string) (string, bool) {
		for _, e := range env {
			if e.Name == name {
				return e.Value, true
			}
		}
		return "", false
	}

	t.Run("emits git identity from agent, overridden by run", func(t *testing.T) {
		agent := &konveyoriov1alpha1.Agent{}
		agent.Spec.GitConfig = gitCfg(gitNameAgent, gitEmailAgent)
		run := &konveyoriov1alpha1.AgentRun{}
		run.Spec.GitConfig = gitCfg(gitNameRun, gitEmailRun)

		env, _, err := r.buildEnvVars(context.Background(), run, agent, "acp-secret")
		if err != nil {
			t.Fatalf("buildEnvVars: %v", err)
		}
		if v, ok := findEnv(env, "KONVEYOR_GIT_AUTHOR_NAME"); !ok || v != gitNameRun {
			t.Errorf("KONVEYOR_GIT_AUTHOR_NAME = %q (present=%v), want %q", v, ok, gitNameRun)
		}
		if v, ok := findEnv(env, "KONVEYOR_GIT_AUTHOR_EMAIL"); !ok || v != gitEmailRun {
			t.Errorf("KONVEYOR_GIT_AUTHOR_EMAIL = %q (present=%v), want %q", v, ok, gitEmailRun)
		}
	})

	t.Run("omits git identity when unset", func(t *testing.T) {
		agent := &konveyoriov1alpha1.Agent{}
		run := &konveyoriov1alpha1.AgentRun{}

		env, _, err := r.buildEnvVars(context.Background(), run, agent, "acp-secret")
		if err != nil {
			t.Fatalf("buildEnvVars: %v", err)
		}
		if _, ok := findEnv(env, "KONVEYOR_GIT_AUTHOR_NAME"); ok {
			t.Error("KONVEYOR_GIT_AUTHOR_NAME should be absent when unset")
		}
		if _, ok := findEnv(env, "KONVEYOR_GIT_AUTHOR_EMAIL"); ok {
			t.Error("KONVEYOR_GIT_AUTHOR_EMAIL should be absent when unset")
		}
	})

	t.Run("drops user-provided git identity env from run.Spec.Env", func(t *testing.T) {
		agent := &konveyoriov1alpha1.Agent{}
		agent.Spec.GitConfig = gitCfg(gitNameAgent, gitEmailAgent)
		run := &konveyoriov1alpha1.AgentRun{}
		// A single-variable override is the worst case: it would forge a
		// hybrid identity (user name + resolved email) if it survived.
		run.Spec.Env = []corev1.EnvVar{
			{Name: "KONVEYOR_GIT_AUTHOR_NAME", Value: "Impostor"},
			{Name: "SOME_OTHER_VAR", Value: "keep-me"},
		}

		env, _, err := r.buildEnvVars(context.Background(), run, agent, "acp-secret")
		if err != nil {
			t.Fatalf("buildEnvVars: %v", err)
		}
		// The managed identity wins and appears exactly once; the user copy
		// is dropped rather than appended after it.
		var names []string
		for _, e := range env {
			if e.Name == "KONVEYOR_GIT_AUTHOR_NAME" {
				names = append(names, e.Value)
			}
		}
		if len(names) != 1 || names[0] != gitNameAgent {
			t.Errorf("KONVEYOR_GIT_AUTHOR_NAME values = %v, want exactly [%q]", names, gitNameAgent)
		}
		// Non-reserved user env is still passed through untouched.
		if v, ok := findEnv(env, "SOME_OTHER_VAR"); !ok || v != "keep-me" {
			t.Errorf("SOME_OTHER_VAR = %q (present=%v), want %q", v, ok, "keep-me")
		}
	})
}
