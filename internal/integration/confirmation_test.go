//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/dongrv/mcp-server-for-mysql/internal/config"
	"github.com/dongrv/mcp-server-for-mysql/internal/tools"
)

func TestMySQLPreviewDoesNotDropThenConfirmationDoes(t *testing.T) {
	service, registry := openIntegrationService(t, config.QuickMode)
	orders := integrationSource(t, registry, "orders")
	executeDirect(t, orders.DB(), "DROP TABLE IF EXISTS integration_preview_target")
	executeDirect(t, orders.DB(), "CREATE TABLE integration_preview_target (id INT PRIMARY KEY)")
	t.Cleanup(func() { executeDirect(t, orders.DB(), "DROP TABLE IF EXISTS integration_preview_target") })

	first, err := service.DropTable(context.Background(), tools.DropTableInput{RequestMeta: tools.RequestMeta{SourceID: "orders"}, Table: "integration_preview_target"})
	if err != nil || first.State != tools.StateConfirmationRequired || first.Preview == nil {
		t.Fatalf("preview drop table = %#v, %v", first, err)
	}
	if !tableExists(t, orders.DB(), "integration_preview_target") {
		t.Fatal("preview executed DROP TABLE")
	}

	_, err = service.DropTable(context.Background(), tools.DropTableInput{RequestMeta: tools.RequestMeta{SourceID: "orders", Confirm: true, PreviewHash: "wrong"}, Table: "integration_preview_target"})
	if !errors.Is(err, tools.ErrPreviewMismatch) {
		t.Fatalf("mismatched confirmation error = %v, want preview mismatch", err)
	}
	if !tableExists(t, orders.DB(), "integration_preview_target") {
		t.Fatal("mismatched confirmation executed DROP TABLE")
	}

	confirmed, err := service.DropTable(context.Background(), tools.DropTableInput{RequestMeta: tools.RequestMeta{SourceID: "orders", Confirm: true, PreviewHash: first.Preview.PreviewHash}, Table: "integration_preview_target"})
	if err != nil || confirmed.State != tools.StateExecuted || confirmed.Execution == nil {
		t.Fatalf("confirmed drop table = %#v, %v", confirmed, err)
	}
	if tableExists(t, orders.DB(), "integration_preview_target") {
		t.Fatal("confirmed DROP TABLE left table behind")
	}
}
