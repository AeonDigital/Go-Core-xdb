package xdb_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/AeonDigital/Go-Core-xdb/pkg/xdb"
	"github.com/AeonDigital/Go-Core-xerrors/pkg/xerrors"
)

func TestLogRepoError(t *testing.T) {
	t.Run("Valida geracao de log completo com query, argumentos e IError500", func(t *testing.T) {
		var buf bytes.Buffer
		slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

		// 1. Abre o banco usando o driver nativo do SQLite
		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatalf("falha ao abrir banco nativo para o teste de log: %v", err)
		}
		defer db.Close()

		ctx := context.Background()
		var code xerrors.ErrorCode = "ERR_UNIQUE_VIOLATION"

		// 2. Cria o erro usando a estrutura exata solicitada
		origErr := xerrors.NewError500(
			xerrors.XERR_PKGCTX,
			xerrors.XERR_ALREADY_EXISTS,
			errors.New("operation failure detected"),
			"op_save",
			"detalhes adicionais",
		)

		query := "INSERT INTO users (name) VALUES (?);"
		args := []any{"Aeon Digital"}

		// 3. Executa a função do logger
		xdb.LogRepoError(ctx, db, code, origErr, query, args)

		// 4. Asserções no log estruturado gerado
		logOutput := buf.String()

		if !strings.Contains(logOutput, `"msg":"operation failure detected"`) {
			t.Error("esperava a mensagem padrão de erro")
		}
		if !strings.Contains(logOutput, `"resource":"sqlite"`) {
			t.Errorf("esperava o recurso correto detectado, recebeu o log: %s", logOutput)
		}
		if !strings.Contains(logOutput, `"sql_query":"INSERT INTO users (name) VALUES (?);"`) {
			t.Error("esperava a query no log")
		}
		if !strings.Contains(logOutput, `"error_cod":"ERR_UNIQUE_VIOLATION"`) {
			t.Error("esperava o código do erro no log")
		}
		if !strings.Contains(logOutput, `"component":`) {
			t.Error("esperava a chave component mapeada no log do IError500")
		}
		if !strings.Contains(logOutput, "operation failure detected") {
			t.Errorf("esperava encontrar a mensagem do erro original no log, recebeu: %s", logOutput)
		}
	})

	t.Run("Valida geracao de log limpo sem query, sem argumentos e sem erro original", func(t *testing.T) {
		var buf bytes.Buffer
		slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

		xdb.LogRepoError(context.Background(), nil, "ERR_UNKNOWN", nil, "", nil)

		logOutput := buf.String()
		if strings.Contains(logOutput, `"sql_query"`) {
			t.Error("não esperava a chave sql_query quando a string for vazia")
		}
		if strings.Contains(logOutput, `"args"`) {
			t.Error("não esperava a chave args quando len(args) == 0")
		}
		if !strings.Contains(logOutput, `"resource":"DB"`) {
			t.Error("esperava recurso padrão 'DB' quando sql.DB for nil")
		}
	})
}
