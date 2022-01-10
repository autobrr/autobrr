package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/autobrr/autobrr/internal/database"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/argon2id"
)

const usage = `usage: autobrrctl --config path <action>

  create-user		 <username>		Create user
  change-password	 <username>		Change password for user
  help						Show this help message
`

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage)
	}
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "path to configuration file")
	flag.Parse()

	if configPath == "" {
		log.Fatal("--config required")
	}

	// open database connection
	db, err := database.OpenSqliteDB(configPath)
	if err != nil {
		log.Fatal("could not open db connection")
	}

	userRepo := database.NewUserRepo(db)

	switch cmd := flag.Arg(0); cmd {
	case "create-user":
		username := flag.Arg(1)
		if username == "" {
			flag.Usage()
			os.Exit(1)
		}

		password, err := readPassword()
		if err != nil {
			log.Fatalf("failed to read password: %v", err)
		}
		hashed, err := argon2id.CreateHash(string(password), argon2id.DefaultParams)
		if err != nil {
			log.Fatalf("failed to hash password: %v", err)
		}

		user := domain.User{
			Username: username,
			Password: hashed,
		}
		if err := userRepo.Store(context.Background(), user); err != nil {
			log.Fatalf("failed to create user: %v", err)
		}
	case "change-password":
		username := flag.Arg(1)
		if username == "" {
			flag.Usage()
			os.Exit(1)
		}

		user, err := userRepo.FindByUsername(context.Background(), username)
		if err != nil {
			log.Fatalf("failed to get user: %v", err)
		}

		if user == nil {
			log.Fatalf("failed to get user: %v", err)
		}

		password, err := readPassword()
		if err != nil {
			log.Fatalf("failed to read password: %v", err)
		}
		hashed, err := argon2id.CreateHash(string(password), argon2id.DefaultParams)
		if err != nil {
			log.Fatalf("failed to hash password: %v", err)
		}

		user.Password = hashed
		if err := userRepo.Store(context.Background(), *user); err != nil {
			log.Fatalf("failed to create user: %v", err)
		}
	default:
		flag.Usage()
		if cmd != "help" {
			os.Exit(1)
		}
	}
}

func readPassword() ([]byte, error) {
	var password []byte
	var err error
	fd := int(os.Stdin.Fd())

	if terminal.IsTerminal(fd) {
		fmt.Printf("Password: ")
		password, err = terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return nil, err
		}
		fmt.Printf("\n")
	} else {
		//fmt.Fprintf(os.Stderr, "warning: Reading password from stdin.\n")
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				log.Fatalf("failed to read password from stdin: %v", err)
			}
			log.Fatalf("failed to read password from stdin: stdin is empty %v", err)
		}
		password = scanner.Bytes()

		if len(password) == 0 {
			return nil, fmt.Errorf("zero length password")
		}
	}

	return password, nil
}
