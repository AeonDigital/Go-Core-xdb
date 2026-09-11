package xdb_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/AeonDigital/Go-Core-xdb/pkg/xdb"
)

// 1. Criamos tipos que implementam as interfaces básicas do driver nativo do Go
type mockDriver struct{}

func (m *mockDriver) Open(name string) (driver.Conn, error) {
	return &mockConn{}, nil
}

type mockConn struct{}

func (m *mockConn) Prepare(query string) (driver.Stmt, error) { return &mockStmt{}, nil }
func (m *mockConn) Close() error                              { return nil }
func (m *mockConn) Begin() (driver.Tx, error)                 { return nil, nil }

// Execer é a interface que o db.Exec chama internamente
func (m *mockConn) Exec(query string, args []driver.Value) (driver.Result, error) {
	return &mockResult{}, nil // Sucesso absoluto em todos os PRAGMAs!
}

// Ping é a interface que o db.PingContext chama internamente.
// Aqui nós forçamos o erro de propósito!
func (m *mockConn) Ping(ctx context.Context) error {
	return fmt.Errorf("erro forcado de ping para cobertura")
}

type mockStmt struct{}

func (m *mockStmt) Close() error                                    { return nil }
func (m *mockStmt) NumInput() int                                   { return 0 }
func (m *mockStmt) Exec(args []driver.Value) (driver.Result, error) { return &mockResult{}, nil }
func (m *mockStmt) Query(args []driver.Value) (driver.Rows, error)  { return nil, nil }

type mockResult struct{}

func (m *mockResult) LastInsertId() (int64, error) { return 0, nil }
func (m *mockResult) RowsAffected() (int64, error) { return 0, nil }

// 2. Registramos o nosso driver mock com um nome customizado no init do arquivo de testes
func init() {
	sql.Register("sqlite_mock_ping_error", &mockDriver{})
}

// TestNewDBConfig valida a inicialização e os valores default.
func TestNewDBConfig(t *testing.T) {
	driver := "sqlite"
	dsn := ""
	migrationsDir := "./migrations"

	cfg := xdb.NewDBConfig(driver, dsn, migrationsDir, "", "", "", "")

	if cfg.Driver != "sqlite" {
		t.Errorf("esperava driver 'sqlite', recebeu '%s'", cfg.Driver)
	}
	if cfg.MaxOpenConnections != 1 {
		t.Errorf("esperava MaxOpenConnections 1, recebeu %d", cfg.MaxOpenConnections)
	}
	if cfg.SQLite.Pragma["journal_mode"] != "WAL" {
		t.Errorf("esperava pragma journal_mode 'WAL', recebeu '%s'", cfg.SQLite.Pragma["journal_mode"])
	}
}

// TestCheckConfiguration cobre os diferentes cenários de criação da string de conexão (antigo buildDSN).
func TestCheckConfiguration(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		inputCfg    xdb.DBConfig
		expectedDSN string
	}{
		{
			name: "DSN preexistente nao deve ser alterada",
			inputCfg: xdb.DBConfig{
				DSN: "file:custom.db?cache=shared",
				SQLite: xdb.SQLiteConfig{
					Mode: "file:",
				},
			},
			expectedDSN: "file:custom.db?cache=shared",
		},
		{
			name: "Memoria pura",
			inputCfg: xdb.DBConfig{
				SQLite: xdb.SQLiteConfig{Mode: ":memory:"},
			},
			expectedDSN: "file::memory:?_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)",
		},
		{
			name: "Memoria com query string",
			inputCfg: xdb.DBConfig{
				SQLite: xdb.SQLiteConfig{
					Mode:        ":memory:",
					QueryString: "cache=shared",
				},
			},
			expectedDSN: "file::memory:?cache=shared&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)",
		},
		{
			name: "Arquivo em disco valido",
			inputCfg: xdb.DBConfig{
				SQLite: xdb.SQLiteConfig{
					Dir:      tmpDir,
					FileName: "local.db",
				},
			},
			expectedDSN: "file:" + filepath.ToSlash(filepath.Join(tmpDir, "local.db")) + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)",
		},
		{
			name: "Arquivo customizado com query string",
			inputCfg: xdb.DBConfig{
				SQLite: xdb.SQLiteConfig{
					Dir:         tmpDir,
					FileName:    "test.db",
					QueryString: "mode=ro",
				},
			},
			expectedDSN: "file:" + filepath.ToSlash(filepath.Join(tmpDir, "test.db")) + "?mode=ro&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)",
		},
		{
			name: "Memoria com file::memory: e query string",
			inputCfg: xdb.DBConfig{
				SQLite: xdb.SQLiteConfig{
					Mode:        "file::memory:",
					QueryString: "cache=shared",
				},
			},
			expectedDSN: "file::memory:?cache=shared&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)",
		},
		{
			name: "Campos vazios nao geram DSN",
			inputCfg: xdb.DBConfig{
				SQLite: xdb.SQLiteConfig{
					Dir:      "",
					FileName: "",
				},
			},
			expectedDSN: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.inputCfg.CheckConfiguration()
			if err != nil {
				t.Fatalf("erro inesperado em CheckConfiguration: %v", err)
			}

			if tt.inputCfg.DSN != tt.expectedDSN {
				t.Errorf("\nesperava: %s\nrecebeu:  %s", tt.expectedDSN, tt.inputCfg.DSN)
			}

			// Se a DSN antiga foi informada, garante que limpou os campos do SQLite
			if strings.Contains(tt.name, "preexistente") && tt.inputCfg.SQLite.Mode != "" {
				t.Error("esperava que os campos SQLite fossem limpos se DSN já existisse")
			}
		})
	}
}

// TestInitDataBaseConnection testa o fluxo de sucesso e as ramificações de erro.
func TestInitDataBaseConnection(t *testing.T) {
	ctx := context.Background()

	t.Run("Erro no sql.Open se o driver for invalido", func(t *testing.T) {
		cfg := xdb.DBConfig{
			Driver: "driver_invalido_para_forcar_erro",
			DSN:    ":memory:",
		}

		err := cfg.InitDataBaseConnection(ctx)
		if err == nil {
			t.Error("esperava erro ao abrir conexao com driver invalido")
		}
	})

	t.Run("Compilar pragmas default antes de abrir a conexao", func(t *testing.T) {
		cfg := xdb.DBConfig{
			Driver: "sqlite",
			SQLite: xdb.SQLiteConfig{
				Mode: ":memory:",
			},
		}

		if err := cfg.CheckConfiguration(); err != nil {
			t.Fatalf("erro inesperado ao compilar configuracao: %v", err)
		}

		err := cfg.InitDataBaseConnection(ctx)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		defer cfg.DB.Close()

		if cfg.SQLite.Pragma == nil {
			t.Error("esperava que o mapa Pragma tivesse sido inicializado")
		}
		if cfg.SQLite.Pragma["journal_mode"] != "WAL" {
			t.Errorf("esperava defaults aplicados, recebeu %v", cfg.SQLite.Pragma["journal_mode"])
		}
	})

	t.Run("Erro ao abrir conexao com pragma invalido", func(t *testing.T) {
		cfg := xdb.DBConfig{
			Driver: "sqlite",
			SQLite: xdb.SQLiteConfig{
				Mode: ":memory:",
				Pragma: map[string]string{
					"journal_mode": "WAL; CREATE TABLE );",
				},
			},
		}

		if err := cfg.CheckConfiguration(); err != nil {
			t.Fatalf("erro inesperado ao compilar configuracao: %v", err)
		}

		err := cfg.InitDataBaseConnection(ctx)
		if err == nil {
			if cfg.DB != nil {
				cfg.DB.Close()
			}
			t.Error("esperava erro de execucao do pragma invalido")
		}
	})

	t.Run("Erro no PingContext simulado por driver mock", func(t *testing.T) {
		cfg := xdb.DBConfig{
			Driver: "sqlite_mock_ping_error",
			DSN:    ":memory:",
			SQLite: xdb.SQLiteConfig{
				Pragma: map[string]string{
					"journal_mode": "WAL",
				},
			},
		}

		err := cfg.InitDataBaseConnection(ctx)
		if err == nil {
			if cfg.DB != nil {
				cfg.DB.Close()
			}
			t.Error("esperava erro no ping do banco")
		}

		if err != nil && !strings.Contains(err.Error(), "liveness ping execution timeout boundary breached") {
			t.Errorf("recebeu um erro diferente do esperado: %v", err)
		}
	})
}

// TestRunMigrations valida todas as ramificações de erro e fluxos lógicos de migração.
func TestRunMigrations(t *testing.T) {
	ctx := context.Background()

	newMockDB := func(t *testing.T) *sql.DB {
		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatalf("falha ao abrir banco em memoria para teste: %v", err)
		}
		return db
	}

	t.Run("Erro se DB for nil", func(t *testing.T) {
		cfg := xdb.DBConfig{MigrationsDirPath: "./migrations"}
		err := cfg.RunMigrations(ctx)
		if err == nil || !strings.Contains(err.Error(), "nil pointer value not allowed") {
			t.Errorf("esperava erro de banco nil, recebeu: %v", err)
		}
	})

	t.Run("Erro se MigrationsDirPath for vazio", func(t *testing.T) {
		db := newMockDB(t)
		defer db.Close()

		cfg := xdb.DBConfig{
			DB:                db,
			MigrationsDirPath: "   ",
		}

		err := cfg.RunMigrations(ctx)
		if err == nil || !strings.Contains(err.Error(), "empty string value not allowed") {
			t.Errorf("esperava erro de diretorio vazio, recebeu: %v", err)
		}
	})

	t.Run("Erro se o caminho fornecido for um arquivo e nao um diretorio", func(t *testing.T) {
		db := newMockDB(t)
		defer db.Close()

		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "dummy.txt")
		err := os.WriteFile(filePath, []byte("conteudo"), 0o644)
		if err != nil {
			t.Fatal(err)
		}

		cfg := xdb.DBConfig{
			DB:                db,
			MigrationsDirPath: filePath,
		}

		err = cfg.RunMigrations(ctx)
		if err == nil || !strings.Contains(err.Error(), "type mismatch restriction violated") {
			t.Errorf("esperava erro de caminho sendo arquivo, recebeu: %v", err)
		}
	})

	t.Run("Erro se falhar ao ler o diretorio por falta de permissao", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Ignorando teste de permissão POSIX 0000 no Windows")
		}
		db := newMockDB(t)
		defer db.Close()

		tmpDir := t.TempDir()
		permDir := filepath.Join(tmpDir, "no_read_dir")
		if err := os.Mkdir(permDir, 0o000); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chmod(permDir, 0o755) }()

		cfg := xdb.DBConfig{
			DB:                db,
			MigrationsDirPath: permDir,
		}

		err := cfg.RunMigrations(ctx)
		if err == nil || !strings.Contains(err.Error(), "target resource currently unavailable") {
			t.Errorf("esperava erro ao ler diretorio protegido, recebeu: %v", err)
		}
	})

	t.Run("Retorna nil se nao houver nenhum arquivo de migracao valido", func(t *testing.T) {
		db := newMockDB(t)
		defer db.Close()

		tmpDir := t.TempDir()

		cfg := xdb.DBConfig{
			DB:                db,
			MigrationsDirPath: tmpDir,
		}

		err := cfg.RunMigrations(ctx)
		if err != nil {
			t.Errorf("nao esperava erro quando len(migrationFiles) == 0, recebeu: %v", err)
		}
	})

	t.Run("Erro se um arquivo SQL sumir ou falhar a leitura no meio do processo", func(t *testing.T) {
		db := newMockDB(t)
		defer db.Close()

		tmpDir := t.TempDir()

		targetInexistente := filepath.Join(tmpDir, "nao_existo.sql")
		symlinkPath := filepath.Join(tmpDir, "01_broken_link.sql")
		if err := os.Symlink(targetInexistente, symlinkPath); err != nil {
			if err := os.WriteFile(symlinkPath, []byte("SELECT 1;"), 0o000); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.Chmod(symlinkPath, 0o644) }()
		}

		cfg := xdb.DBConfig{
			DB:                db,
			MigrationsDirPath: tmpDir,
		}

		err := cfg.RunMigrations(ctx)
		if err == nil || !strings.Contains(err.Error(), "target resource is corrupted") {
			t.Errorf("esperava erro de leitura do arquivo individual, recebeu: %v", err)
		}
	})

	t.Run("Pular arquivo de migracao se ele estiver vazio ou apenas com espacos", func(t *testing.T) {
		db := newMockDB(t)
		defer db.Close()

		tmpDir := t.TempDir()

		if err := os.WriteFile(filepath.Join(tmpDir, "01_empty.sql"), []byte("   \n   "), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "02_valid.sql"), []byte("CREATE TABLE logs (id INT);"), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg := xdb.DBConfig{
			DB:                db,
			MigrationsDirPath: tmpDir,
		}

		err := cfg.RunMigrations(ctx)
		if err != nil {
			t.Fatalf("nao esperava erro ao processar migracao vazia com continue, recebeu: %v", err)
		}

		_, err = db.Exec("INSERT INTO logs (id) VALUES (1);")
		if err != nil {
			t.Errorf("tabela logs deveria ter sido criada, indicando que o continue funcionou: %v", err)
		}
	})

	t.Run("Erro de sintaxe SQL na execucao da migracao", func(t *testing.T) {
		db := newMockDB(t)
		defer db.Close()

		tmpDir := t.TempDir()
		sqlInvalido := "INSERT INTO;"

		if err := os.WriteFile(filepath.Join(tmpDir, "01_broken.sql"), []byte(sqlInvalido), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg := xdb.DBConfig{
			DB:                db,
			MigrationsDirPath: tmpDir,
		}

		err := cfg.RunMigrations(ctx)
		if err == nil || !strings.Contains(err.Error(), "failed transaction command context lifecycle validation execution block") {
			t.Errorf("esperava erro de execucao de script SQL invalido, recebeu: %v", err)
		}
	})

	t.Run("Erro se o diretorio de migracoes nao existir em disco", func(t *testing.T) {
		db := newMockDB(t)
		defer db.Close()

		tmpDir := t.TempDir()
		caminhoInexistente := filepath.Join(tmpDir, "pasta_totalmente_fantasma_123")

		cfg := xdb.DBConfig{
			DB:                db,
			MigrationsDirPath: caminhoInexistente,
		}

		err := cfg.RunMigrations(ctx)
		if err == nil || !strings.Contains(err.Error(), "target system resource or path not found") {
			t.Errorf("esperava erro de diretorio inexistente, recebeu: %v", err)
		}
	})
}
