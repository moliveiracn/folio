package main

import (
	"strconv"

	"github.com/moliveiracn/folio/internal/config"

	"crypto/rand"
	"database/sql"
	"flag"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "serve":
		runServe(args[1:])
	case "init":
		runInit(args[1:])
	case "passwd":
		runPasswd(args[1:])
	default:
		fmt.Printf("Unknown command: %s\n", args[0])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: folio <command> [options]")
	fmt.Println("commands: serve, init, passwd")
}

// --- Subcommand Handlers ---

func runServe(args []string) {
	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	dataDirFlag := serveCmd.String("data", "./data", "data directory path")
	serveCmd.Parse(args)

	//read config
	configPath := filepath.Join(*dataDirFlag, "config.yaml")
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		fmt.Println("Have you run: folio init --data", *dataDirFlag)
		os.Exit(1)
	}
	addr := ":" + strconv.Itoa(cfg.Port)

	fmt.Printf("Starting folio on %s...\n", addr)
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<h1>Folio is running</h1><p>This is a static HTML page.</p>"))
	})

	if err := http.ListenAndServe(addr, r); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}

func runInit(args []string) {
	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	dataDirFlag := initCmd.String("data", "./data", "data directory path")
	passwdFlag := initCmd.String("passwd", "", "Password")
	portFlag := initCmd.String("port", "8080", "HTTP port")
	initCmd.Parse(args)

	fmt.Println("Initializing folio...")

	// check if config file already exist
	configPath := filepath.Join(*dataDirFlag, "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		fmt.Println("folio is already initialized.")
		fmt.Printf("To change the password run: folio passwd --data %s\n", *dataDirFlag)
		os.Exit(0)
	}

	// create folder structure
	dirs := []string{
		*dataDirFlag,
		filepath.Join(*dataDirFlag, "books"),
		filepath.Join(*dataDirFlag, "plugins"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			fmt.Printf("error: could not create directory %s: %v\n", d, err)
			os.Exit(1)
		}
	}

	// create an empty database
	dbPath := filepath.Join(*dataDirFlag, "folio.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Printf("error: could not open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// verify the connection actually works
	if err := db.Ping(); err != nil {
		fmt.Printf("error: could not connect to database: %v\n", err)
		os.Exit(1)
	}

	// migration (nshtw)
	// TODO: create a function/helper(?) + create tracker table
	migrationSQL, err := os.ReadFile(
		filepath.Join("internal", "repo", "migrations", "001_init.sql"),
	)
	if err != nil {
		fmt.Printf("error: could not read migration file: %v\n", err)
		os.Exit(1)
	}
	if _, err := db.Exec(string(migrationSQL)); err != nil {
		fmt.Printf("error: migration failed: %v\n", err)
		os.Exit(1)
	}

	// generate password or use passwdFlag
	password := *passwdFlag
	if password == "" {
		password = generatePassword(12)
		fmt.Println("Using generated password...")
	}

	// hash passwd bcrypt
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("error: could not hash password: %v\n", err)
		os.Exit(1)
	}

	// write config file
	config := fmt.Sprintf(
		"port: %s\ndata_dir: %s\npassword_hash: %s\nlog_level: info\nmax_upload_mb: 200",
		*portFlag, *dataDirFlag, string(hash),
	)
	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		fmt.Printf("error: could not write config.yaml: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("folio initialized.")
	fmt.Println()
	fmt.Printf("  Data dir : %s\n", *dataDirFlag)
	fmt.Printf("  Database : %s\n", dbPath)
	fmt.Printf("  Port     : %s\n", *portFlag)
	fmt.Printf("  Password : %s\n", password)
	fmt.Println()
	fmt.Println("Keep this password safe. It will not be shown again.")
	fmt.Printf("Run: folio serve --data %s\n", *dataDirFlag)
}

func runPasswd(args []string) {
	passwdCmd := flag.NewFlagSet("passwd", flag.ExitOnError)
	passwdCmd.Parse(args)

	var password string
	remainingArgs := passwdCmd.Args()

	if len(remainingArgs) == 0 {
		password = generatePassword(12)
		fmt.Println("Generated new random password...")
	} else {
		password = remainingArgs[0]
		fmt.Println("Updating password with provided value...")
	}
	fmt.Printf("New password: %s\n", password)
}

// --- Utilities ---

func generatePassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	res := make([]byte, length)
	for i := range res {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		res[i] = charset[num.Int64()]
	}
	return string(res)
}
