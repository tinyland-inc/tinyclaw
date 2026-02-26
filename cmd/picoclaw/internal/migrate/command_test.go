package migrate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMigrateCommand(t *testing.T) {
	cmd := NewMigrateCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "migrate", cmd.Use)
	assert.Equal(t, "Migrate configuration between formats", cmd.Short)

	assert.Empty(t, cmd.Aliases)

	assert.True(t, cmd.HasExample())
	assert.True(t, cmd.HasSubCommands())

	assert.Nil(t, cmd.Run)
	assert.NotNil(t, cmd.RunE)

	assert.Nil(t, cmd.PersistentPreRun)
	assert.Nil(t, cmd.PersistentPostRun)

	assert.True(t, cmd.HasFlags())

	// Legacy flags on root migrate command
	assert.NotNil(t, cmd.Flags().Lookup("dry-run"))
	assert.NotNil(t, cmd.Flags().Lookup("force"))
}

func TestNewMigrateCommand_OpenclawSubcommand(t *testing.T) {
	cmd := NewMigrateCommand()

	openclaw, _, err := cmd.Find([]string{"openclaw"})
	require.NoError(t, err)
	require.NotNil(t, openclaw)

	assert.Equal(t, "openclaw", openclaw.Use)
	assert.NotNil(t, openclaw.Flags().Lookup("dry-run"))
	assert.NotNil(t, openclaw.Flags().Lookup("refresh"))
	assert.NotNil(t, openclaw.Flags().Lookup("config-only"))
	assert.NotNil(t, openclaw.Flags().Lookup("workspace-only"))
	assert.NotNil(t, openclaw.Flags().Lookup("force"))
	assert.NotNil(t, openclaw.Flags().Lookup("openclaw-home"))
	assert.NotNil(t, openclaw.Flags().Lookup("picoclaw-home"))
}

func TestNewMigrateCommand_ToDhallSubcommand(t *testing.T) {
	cmd := NewMigrateCommand()

	toDhall, _, err := cmd.Find([]string{"to-dhall"})
	require.NoError(t, err)
	require.NotNil(t, toDhall)

	assert.Equal(t, "to-dhall", toDhall.Use)
	assert.NotNil(t, toDhall.Flags().Lookup("config"))
	assert.NotNil(t, toDhall.Flags().Lookup("output"))
	assert.NotNil(t, toDhall.Flags().Lookup("dry-run"))
	assert.NotNil(t, toDhall.Flags().Lookup("force"))
}
