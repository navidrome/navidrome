package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	email      string
	libraryIds []int
	name       string

	removeEmail    bool
	removeName     bool
	setAdmin       bool
	setPassword    bool
	setRegularUser bool
)

func init() {
	rootCmd.AddCommand(userRoot)

	userCreateCommand.Flags().StringVarP(&userID, "username", "u", "", "username")

	userCreateCommand.Flags().StringVarP(&email, "email", "e", "", "New user email")
	userCreateCommand.Flags().IntSliceVarP(&libraryIds, "library-ids", "i", []int{}, "Comma-separated list of library IDs. Set the user's accessible libraries. If empty, the user can access all libraries. This is incompatible with admin, as admin can always access all libraries")

	userCreateCommand.Flags().BoolVarP(&setAdmin, "admin", "a", false, "If set, make the user an admin. This user will have access to every library")
	userCreateCommand.Flags().StringVar(&name, "name", "", "New user's name (this is separate from username used to log in)")

	_ = userCreateCommand.MarkFlagRequired("username")

	userRoot.AddCommand(userCreateCommand)

	userDeleteCommand.Flags().StringVarP(&userID, "user", "u", "", "username or id")
	_ = userDeleteCommand.MarkFlagRequired("user")
	userRoot.AddCommand(userDeleteCommand)

	userEditCommand.Flags().StringVarP(&userID, "user", "u", "", "username or id")

	userEditCommand.Flags().BoolVar(&setAdmin, "set-admin", false, "If set, make the user an admin")
	userEditCommand.Flags().BoolVar(&setRegularUser, "set-regular", false, "If set, make the user a non-admin")
	userEditCommand.MarkFlagsMutuallyExclusive("set-admin", "set-regular")

	userEditCommand.Flags().BoolVar(&removeEmail, "remove-email", false, "If set, clear the user's email")
	userEditCommand.Flags().StringVarP(&email, "email", "e", "", "New user email")
	userEditCommand.MarkFlagsMutuallyExclusive("email", "remove-email")

	userEditCommand.Flags().BoolVar(&removeName, "remove-name", false, "If set, clear the user's name")
	userEditCommand.Flags().StringVar(&name, "name", "", "New user name (this is separate from username used to log in)")
	userEditCommand.MarkFlagsMutuallyExclusive("name", "remove-name")

	userEditCommand.Flags().BoolVar(&setPassword, "set-password", false, "If set, the user's new password will be prompted on the CLI")

	userEditCommand.Flags().IntSliceVarP(&libraryIds, "library-ids", "i", []int{}, "Comma-separated list of library IDs. Set the user's accessible libraries by id")

	_ = userEditCommand.MarkFlagRequired("user")
	userRoot.AddCommand(userEditCommand)

	userListCommand.Flags().StringVarP(&outputFormat, "format", "f", "csv", "output format [supported values: csv, json]")
	userRoot.AddCommand(userListCommand)
}

var (
	userRoot = &cobra.Command{
		Use:   "user",
		Short: "Administer users",
		Long:  "Create, delete, list, or update users",
	}

	userCreateCommand = &cobra.Command{
		Use:     "create",
		Aliases: []string{"c"},
		Short:   "Create a new user",
		Run: func(cmd *cobra.Command, args []string) {
			runCreateUser(cmd.Context())
		},
	}

	userDeleteCommand = &cobra.Command{
		Use:     "delete",
		Aliases: []string{"d"},
		Short:   "Deletes an existing user",
		Run: func(cmd *cobra.Command, args []string) {
			runDeleteUser(cmd.Context())
		},
	}

	userEditCommand = &cobra.Command{
		Use:     "edit",
		Aliases: []string{"e"},
		Short:   "Edit a user",
		Long:    "Edit the password, admin status, and/or library access",
		Run: func(cmd *cobra.Command, args []string) {
			runUserEdit(cmd.Context())
		},
	}

	userListCommand = &cobra.Command{
		Use:   "list",
		Short: "List users",
		Run: func(cmd *cobra.Command, args []string) {
			runUserList(cmd.Context())
		},
	}
)

func promptPassword() string {
	for {
		fmt.Print("Enter new password (press enter with no password to cancel): ")
		// This cast is necessary for some platforms
		password, err := term.ReadPassword(int(syscall.Stdin)) //nolint:unconvert

		if err != nil {
			log.Fatal("Error getting password", err)
		}

		fmt.Print("\nConfirm new password (press enter with no password to cancel): ")
		confirmation, err := term.ReadPassword(int(syscall.Stdin)) //nolint:unconvert

		if err != nil {
			log.Fatal("Error getting password confirmation", err)
		}

		// clear the line.
		fmt.Println()

		pass := string(password)
		confirm := string(confirmation)

		if pass == "" {
			return ""
		}

		if pass == confirm {
			return pass
		}

		fmt.Println("Password and password confirmation do not match")
	}
}

func libraryError(libraries model.Libraries) error {
	ids := make([]int, len(libraries))
	for idx, library := range libraries {
		ids[idx] = library.ID
	}
	return fmt.Errorf("not all available libraries found. Requested ids: %v, Found libraries: %v", libraryIds, ids)
}

func runCreateUser(ctx context.Context) {
	password := promptPassword()
	if password == "" {
		log.Fatal("Empty password provided, user creation cancelled")
	}

	user := model.User{
		UserName:    userID,
		Email:       email,
		Name:        name,
		IsAdmin:     setAdmin,
		NewPassword: password,
	}

	if user.Name == "" {
		user.Name = userID
	}

	ds, ctx := getAdminContext(ctx)

	err := ds.WithTx(func(tx model.DataStore) error {
		existingUser, err := tx.User(ctx).FindByUsername(userID)
		if existingUser != nil {
			return fmt.Errorf("existing user '%s'", userID)
		}

		if err != nil && !errors.Is(err, model.ErrNotFound) {
			return fmt.Errorf("failed to check existing username: %w", err)
		}

		if len(libraryIds) > 0 && !setAdmin {
			user.Libraries, err = tx.Library(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"id": libraryIds}})
			if err != nil {
				return err
			}

			if len(user.Libraries) != len(libraryIds) {
				return libraryError(user.Libraries)
			}
		} else {
			user.Libraries, err = tx.Library(ctx).GetAll()
			if err != nil {
				return err
			}
		}

		err = tx.User(ctx).Put(&user)
		if err != nil {
			return err
		}

		updatedIds := make([]int, len(user.Libraries))
		for idx, lib := range user.Libraries {
			updatedIds[idx] = lib.ID
		}

		err = tx.User(ctx).SetUserLibraries(user.ID, updatedIds)
		return err
	})

	if err != nil {
		log.Fatal(ctx, err)
	}

	log.Info(ctx, "Successfully created user", "id", user.ID, "username", user.UserName)
}

func runDeleteUser(ctx context.Context) {
	ds, ctx := getAdminContext(ctx)

	var err error
	var user *model.User

	err = ds.WithTx(func(tx model.DataStore) error {
		count, err := tx.User(ctx).CountAll()
		if err != nil {
			return err
		}

		if count == 1 {
			return errors.New("refusing to delete the last user")
		}

		user, err = getUser(ctx, userID, tx)
		if err != nil {
			return err
		}

		return tx.User(ctx).Delete(user.ID)
	})

	if err != nil {
		log.Fatal(ctx, "Failed to delete user", err)
	}

	log.Info(ctx, "Deleted user", "username", user.UserName)
}

func runUserEdit(ctx context.Context) {
	ds, ctx := getAdminContext(ctx)

	var err error
	var user *model.User
	changes := []string{}

	err = ds.WithTx(func(tx model.DataStore) error {
		var newLibraries model.Libraries

		user, err = getUser(ctx, userID, tx)
		if err != nil {
			return err
		}

		if len(libraryIds) > 0 && !setAdmin {
			libraries, err := tx.Library(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"id": libraryIds}})

			if err != nil {
				return err
			}

			if len(libraries) != len(libraryIds) {
				return libraryError(libraries)
			}

			newLibraries = libraries
			changes = append(changes, "updated library ids")
		}

		if setAdmin && !user.IsAdmin {
			libraries, err := tx.Library(ctx).GetAll()
			if err != nil {
				return err
			}

			user.IsAdmin = true
			user.Libraries = libraries
			changes = append(changes, "set admin")

			newLibraries = libraries
		}

		if setRegularUser && user.IsAdmin {
			user.IsAdmin = false
			changes = append(changes, "set regular user")
		}

		if setPassword {
			password := promptPassword()

			if password != "" {
				user.NewPassword = password
				changes = append(changes, "updated password")
			}
		}

		if email != "" && email != user.Email {
			user.Email = email
			changes = append(changes, "updated email")
		} else if removeEmail && user.Email != "" {
			user.Email = ""
			changes = append(changes, "removed email")
		}

		if name != "" && name != user.Name {
			user.Name = name
			changes = append(changes, "updated name")
		} else if removeName && user.Name != "" {
			user.Name = ""
			changes = append(changes, "removed name")
		}

		if len(changes) == 0 {
			return nil
		}

		err := tx.User(ctx).Put(user)
		if err != nil {
			return err
		}

		if len(newLibraries) > 0 {
			updatedIds := make([]int, len(newLibraries))
			for idx, lib := range newLibraries {
				updatedIds[idx] = lib.ID
			}

			err := tx.User(ctx).SetUserLibraries(user.ID, updatedIds)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		log.Fatal(ctx, "Failed to update user", err)
	}

	if len(changes) == 0 {
		log.Info(ctx, "No changes for user", "user", user.UserName)
	} else {
		log.Info(ctx, "Updated user", "user", user.UserName, "changes", strings.Join(changes, ", "))
	}
}

type displayLibrary struct {
	ID   int    `json:"id"`
	Path string `json:"path"`
}

type displayUser struct {
	Id         string           `json:"id"`
	Username   string           `json:"username"`
	Name       string           `json:"name"`
	Email      string           `json:"email"`
	Admin      bool             `json:"admin"`
	CreatedAt  time.Time        `json:"createdAt"`
	UpdatedAt  time.Time        `json:"updatedAt"`
	LastAccess *time.Time       `json:"lastAccess"`
	LastLogin  *time.Time       `json:"lastLogin"`
	Libraries  []displayLibrary `json:"libraries"`
}

func runUserList(ctx context.Context) {
	if outputFormat != "csv" && outputFormat != "json" {
		log.Fatal("Invalid output format. Must be one of csv, json", "format", outputFormat)
	}

	ds, ctx := getAdminContext(ctx)

	users, err := ds.User(ctx).ReadAll()
	if err != nil {
		log.Fatal(ctx, "Failed to retrieve users", err)
	}

	userList := users.(model.Users)

	if outputFormat == "csv" {
		w := csv.NewWriter(os.Stdout)
		_ = w.Write([]string{
			"user id",
			"username",
			"user's name",
			"user email",
			"admin",
			"created at",
			"updated at",
			"last access",
			"last login",
			"libraries",
		})
		for _, user := range userList {
			paths := make([]string, len(user.Libraries))

			for idx, library := range user.Libraries {
				paths[idx] = fmt.Sprintf("%d:%s", library.ID, library.Path)
			}

			var lastAccess, lastLogin string

			if user.LastAccessAt != nil {
				lastAccess = user.LastAccessAt.Format(time.RFC3339Nano)
			} else {
				lastAccess = "never"
			}

			if user.LastLoginAt != nil {
				lastLogin = user.LastLoginAt.Format(time.RFC3339Nano)
			} else {
				lastLogin = "never"
			}

			_ = w.Write([]string{
				user.ID,
				user.UserName,
				user.Name,
				user.Email,
				strconv.FormatBool(user.IsAdmin),
				user.CreatedAt.Format(time.RFC3339Nano),
				user.UpdatedAt.Format(time.RFC3339Nano),
				lastAccess,
				lastLogin,
				fmt.Sprintf("'%s'", strings.Join(paths, "|")),
			})
		}
		w.Flush()
	} else {
		users := make([]displayUser, len(userList))
		for idx, user := range userList {
			paths := make([]displayLibrary, len(user.Libraries))

			for idx, library := range user.Libraries {
				paths[idx].ID = library.ID
				paths[idx].Path = library.Path
			}

			users[idx].Id = user.ID
			users[idx].Username = user.UserName
			users[idx].Name = user.Name
			users[idx].Email = user.Email
			users[idx].Admin = user.IsAdmin
			users[idx].CreatedAt = user.CreatedAt
			users[idx].UpdatedAt = user.UpdatedAt
			users[idx].LastAccess = user.LastAccessAt
			users[idx].LastLogin = user.LastLoginAt
			users[idx].Libraries = paths
		}

		j, _ := json.Marshal(users)
		fmt.Printf("%s\n", j)
	}
}
