package product

import "testing"

func TestHostedCommunityIsRejected(t *testing.T) {
	settings := Settings{Edition: EditionCommunity, Deployment: DeploymentHosted, BindAddress: "0.0.0.0", OwnerID: "user"}
	if err := settings.Validate(); err == nil {
		t.Fatal("expected hosted community configuration to be rejected")
	}
}

func TestHostedDeploymentRequiresHTTPSWhenPublicURLIsSet(t *testing.T) {
	settings := Settings{Edition: EditionCloud, Deployment: DeploymentHosted, BindAddress: "0.0.0.0", OwnerID: "user", PublicURL: "http://eraser.example", AuthIssuer: "https://id.example", AuthAudience: "eraser", DatabaseURL: "postgres://eraser@example/eraser"}
	if err := settings.Validate(); err == nil {
		t.Fatal("expected an insecure hosted public URL to be rejected")
	}
	settings.PublicURL = "https://eraser.example"
	if err := settings.Validate(); err != nil {
		t.Fatalf("expected valid hosted settings: %v", err)
	}
}

func TestHostedDeploymentRequiresOIDCAndExplicitOwner(t *testing.T) {
	settings := Settings{Edition: EditionCloud, Deployment: DeploymentHosted, BindAddress: "0.0.0.0", OwnerID: "local"}
	if err := settings.Validate(); err == nil {
		t.Fatal("expected default local owner to be rejected")
	}
	settings.OwnerID = "owner-1"
	if err := settings.Validate(); err == nil {
		t.Fatal("expected missing OIDC configuration to be rejected")
	}
	settings.AuthIssuer = "https://id.example"
	settings.AuthAudience = "eraser"
	if err := settings.Validate(); err == nil {
		t.Fatal("expected missing PostgreSQL configuration to be rejected")
	}
}

func TestPublicInfoDoesNotExposeOwnerID(t *testing.T) {
	settings := Settings{Edition: EditionCommunity, Deployment: DeploymentSelfHosted, BindAddress: "127.0.0.1", OwnerID: "private-subject"}
	info := settings.Info()
	if info.Capabilities.ManagedService || !info.Capabilities.SelfHosting || !info.Capabilities.SingleUser {
		t.Fatalf("unexpected capabilities: %+v", info.Capabilities)
	}
}

func TestPostgresRequiresExplicitOwnerInEveryDeployment(t *testing.T) {
	settings := Settings{
		Edition: EditionCommercial, Deployment: DeploymentSelfHosted,
		BindAddress: "127.0.0.1", OwnerID: "local", DatabaseURL: "postgres://eraser@example/eraser",
	}
	if err := settings.Validate(); err == nil {
		t.Fatal("expected PostgreSQL with the default owner to be rejected")
	}
}
