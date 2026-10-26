package git

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yvv4git/git-commit-gen/internal/domain"
	"github.com/yvv4git/git-commit-gen/internal/ports"
)

const (
	testBaseBranch = "main"
	testWorkBranch = "feature"
)

type testRepo struct {
	t *testing.T
}

func newTestRepo(t *testing.T) *testRepo {
	t.Helper()
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Chdir(t.TempDir())

	repo := &testRepo{t: t}
	repo.run("init", "-q", "-b", testBaseBranch)
	repo.run("config", "user.name", "Test")
	repo.run("config", "user.email", "test@example.com")
	repo.run("config", "commit.gpgsign", "false")

	return repo
}

func (r *testRepo) run(args ...string) string {
	r.t.Helper()

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	require.NoError(r.t, err, "git %s: %s", strings.Join(args, " "), output)

	return strings.TrimSpace(string(output))
}

func (r *testRepo) writeFile(name, content string) {
	r.t.Helper()
	require.NoError(r.t, os.WriteFile(name, []byte(content), 0o644))
}

func (r *testRepo) commit(message string) string {
	r.t.Helper()
	r.run("add", "-A")
	r.run("commit", "-q", "-m", message)

	return r.run("rev-parse", "HEAD")
}

func (r *testRepo) createBranch(name string) {
	r.t.Helper()
	r.run("checkout", "-q", "-b", name)
}

func (r *testRepo) checkout(name string) {
	r.t.Helper()
	r.run("checkout", "-q", name)
}

func TestUpdateFirstCommitNoBranchCommits(t *testing.T) {
	repo := newTestRepo(t)
	repo.writeFile("a.txt", "one")
	mainHash := repo.commit("Initial commit")
	repo.createBranch(testWorkBranch)

	client := New()

	_, err := client.UpdateFirstCommit(context.Background(), &ports.UpdateFirstCommitParams{
		CommitMessage: "Generated message",
		BaseBranch:    testBaseBranch,
	})

	require.ErrorIs(t, err, domain.ErrNoCommitsToUpdate)
	require.Equal(t, mainHash, repo.run("rev-parse", testBaseBranch))
	require.Equal(t, mainHash, repo.run("rev-parse", testWorkBranch))
}

func TestUpdateFirstCommitNoBranchCommitsWithLongBase(t *testing.T) {
	repo := newTestRepo(t)
	repo.writeFile("a.txt", "one")
	repo.commit("Initial commit")
	repo.writeFile("b.txt", "two")
	mainHash := repo.commit("Second main commit")
	repo.createBranch(testWorkBranch)

	client := New()

	_, err := client.UpdateFirstCommit(context.Background(), &ports.UpdateFirstCommitParams{
		CommitMessage: "Generated message",
		BaseBranch:    testBaseBranch,
	})

	require.ErrorIs(t, err, domain.ErrNoCommitsToUpdate)
	require.Equal(t, mainHash, repo.run("rev-parse", testBaseBranch))
	require.Equal(t, mainHash, repo.run("rev-parse", testWorkBranch))
}

func TestUpdateFirstCommitBranchBehindBase(t *testing.T) {
	repo := newTestRepo(t)
	repo.writeFile("a.txt", "one")
	branchHash := repo.commit("Initial commit")
	repo.createBranch(testWorkBranch)
	repo.checkout(testBaseBranch)
	repo.writeFile("b.txt", "two")
	mainHash := repo.commit("Second main commit")

	client := New()

	_, err := client.UpdateFirstCommit(context.Background(), &ports.UpdateFirstCommitParams{
		CommitMessage: "Generated message",
		BaseBranch:    testBaseBranch,
	})

	require.ErrorIs(t, err, domain.ErrNoCommitsToUpdate)
	require.Equal(t, mainHash, repo.run("rev-parse", testBaseBranch))
	require.Equal(t, branchHash, repo.run("rev-parse", testWorkBranch))
}

func TestUpdateFirstCommitSingleBranchCommit(t *testing.T) {
	repo := newTestRepo(t)
	repo.writeFile("a.txt", "one")
	mainHash := repo.commit("Initial commit")
	repo.createBranch(testWorkBranch)
	repo.writeFile("b.txt", "two")
	oldHash := repo.commit("Feature commit")
	oldTree := repo.run("rev-parse", "HEAD^{tree}")

	client := New()

	_, err := client.UpdateFirstCommit(context.Background(), &ports.UpdateFirstCommitParams{
		CommitMessage: "Generated message",
		BaseBranch:    testBaseBranch,
	})
	require.NoError(t, err)

	newHash := repo.run("rev-parse", testWorkBranch)
	require.NotEqual(t, oldHash, newHash)
	require.Equal(t, "Generated message", repo.run("log", "-1", "--format=%s", testWorkBranch))
	require.Equal(t, mainHash, repo.run("rev-parse", testWorkBranch+"^"))
	require.Equal(t, oldTree, repo.run("rev-parse", testWorkBranch+"^{tree}"))
	require.Equal(t, mainHash, repo.run("rev-parse", testBaseBranch))
	require.Equal(t, "1", repo.run("rev-list", "--count", testBaseBranch+".."+testWorkBranch))
}

func TestUpdateFirstCommitTwoBranchCommits(t *testing.T) {
	repo := newTestRepo(t)
	repo.writeFile("a.txt", "one")
	mainHash := repo.commit("Initial commit")
	repo.createBranch(testWorkBranch)
	repo.writeFile("b.txt", "two")
	firstOld := repo.commit("First feature commit")
	firstTree := repo.run("rev-parse", firstOld+"^{tree}")
	repo.writeFile("c.txt", "three")
	secondOld := repo.commit("Second feature commit")
	secondTree := repo.run("rev-parse", secondOld+"^{tree}")

	client := New()

	_, err := client.UpdateFirstCommit(context.Background(), &ports.UpdateFirstCommitParams{
		CommitMessage: "Generated message",
		BaseBranch:    testBaseBranch,
	})
	require.NoError(t, err)

	firstNew := repo.run("rev-parse", testWorkBranch+"^")
	secondNew := repo.run("rev-parse", testWorkBranch)
	require.NotEqual(t, firstOld, firstNew)
	require.NotEqual(t, secondOld, secondNew)
	require.Equal(t, "Generated message", repo.run("log", "-1", "--format=%s", firstNew))
	require.Equal(t, "Second feature commit", repo.run("log", "-1", "--format=%s", secondNew))
	require.Equal(t, mainHash, repo.run("rev-parse", firstNew+"^"))
	require.Equal(t, firstTree, repo.run("rev-parse", firstNew+"^{tree}"))
	require.Equal(t, secondTree, repo.run("rev-parse", secondNew+"^{tree}"))
	require.Equal(t, mainHash, repo.run("rev-parse", testBaseBranch))
	require.Equal(t, "2", repo.run("rev-list", "--count", testBaseBranch+".."+testWorkBranch))
}

func TestFirstCommitMessageSingleBranchCommit(t *testing.T) {
	repo := newTestRepo(t)
	repo.writeFile("a.txt", "one")
	repo.commit("Initial commit")
	repo.createBranch(testWorkBranch)
	repo.writeFile("b.txt", "two")
	repo.commit("Feature commit")

	client := New()

	res, err := client.FirstCommitMessage(context.Background(), &ports.FirstCommitMessageParams{
		BaseBranch: testBaseBranch,
	})
	require.NoError(t, err)
	require.Equal(t, "Feature commit", strings.TrimSpace(res.CommitMessage))
}

func TestFirstCommitMessageTwoBranchCommits(t *testing.T) {
	repo := newTestRepo(t)
	repo.writeFile("a.txt", "one")
	repo.commit("Initial commit")
	repo.createBranch(testWorkBranch)
	repo.writeFile("b.txt", "two")
	repo.commit("First feature commit")
	repo.writeFile("c.txt", "three")
	repo.commit("Second feature commit")

	client := New()

	res, err := client.FirstCommitMessage(context.Background(), &ports.FirstCommitMessageParams{
		BaseBranch: testBaseBranch,
	})
	require.NoError(t, err)
	require.Equal(t, "First feature commit", strings.TrimSpace(res.CommitMessage))
}

func TestFirstCommitMessageNoBranchCommits(t *testing.T) {
	repo := newTestRepo(t)
	repo.writeFile("a.txt", "one")
	repo.commit("Initial commit")
	repo.createBranch(testWorkBranch)

	client := New()

	_, err := client.FirstCommitMessage(context.Background(), &ports.FirstCommitMessageParams{
		BaseBranch: testBaseBranch,
	})

	require.ErrorIs(t, err, domain.ErrNoCommitsToUpdate)
}

func TestLoadDiffNoBranchCommits(t *testing.T) {
	repo := newTestRepo(t)
	repo.writeFile("a.txt", "one")
	repo.commit("Initial commit")
	repo.createBranch(testWorkBranch)

	client := New()

	_, err := client.LoadDiff(context.Background(), &ports.LoadDiffParams{BaseBranch: testBaseBranch})

	require.ErrorIs(t, err, domain.ErrNoCommitsToUpdate)
}
