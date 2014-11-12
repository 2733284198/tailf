package tailf_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aybabtme/tailf"
)

func TestCanFollowFile(t *testing.T) { withTempFile(t, canFollowFile) }

func canFollowFile(t *testing.T, filename string, file *os.File) error {

	toWrite := []string{
		"hello,",
		" world!",
	}

	want := strings.Join(toWrite, "")

	follow, err := tailf.Follow(filename, true)
	if err != nil {
		return fmt.Errorf("creating follower: %v", err)
	}

	go func() {
		for _, str := range toWrite {
			log.Printf("writing")
			_, err := file.WriteString(str)
			if err != nil {
				t.Fatalf("failed to write to test file: %v", err)
			}
		}
	}()

	// this should work, without blocking forever
	data := make([]byte, len(want))
	_, err = io.ReadAtLeast(follow, data, len(want))
	if err != nil {
		return err
	}

	// this should block forever
	errc := make(chan error, 1)
	go func() {
		n, err := follow.Read(make([]byte, 1))
		t.Logf("read %d bytes after closing", n)
		errc <- err
	}()

	if err := follow.Close(); err != nil {
		t.Errorf("failed to close follower: %v", err)
	}

	got := string(data)
	if want != got {
		t.Errorf("want %v, got %v", want, got)
	}

	err = <-errc
	if err != io.EOF {
		t.Errorf("expected EOF after closing follower, got %v", err)
	}

	return nil
}

func TestCanFollowFileOverwritten(t *testing.T) { withTempFile(t, canFollowFileOverwritten) }

func canFollowFileOverwritten(t *testing.T, filename string, file *os.File) error {

	toWrite := []string{
		"hello,",
		" world!",
	}
	toWriteAgain := []string{
		"bonjour,",
		" le monde!",
	}

	want := strings.Join(append(toWrite, toWriteAgain...), "")

	follow, err := tailf.Follow(filename, true)
	if err != nil {
		return fmt.Errorf("creating follower: %v", err)
	}

	go func() {
		for _, str := range toWrite {
			log.Printf("writing")
			_, err := file.WriteString(str)
			if err != nil {
				t.Fatalf("failed to write to test file: %v", err)
			}
		}

		if err := os.Remove(filename); err != nil {
			t.Fatalf("couldn't delete file %q: %v", filename, err)
		}

		file, err = os.Create(filename)
		if err != nil {
			t.Fatalf("failed to write to test file: %v", err)
		}
		defer file.Close()
		for _, str := range toWriteAgain {
			log.Printf("writing again")
			_, err := file.WriteString(str)
			if err != nil {
				t.Fatalf("failed to write to test file: %v", err)
			}
		}

	}()

	// this should work, without blocking forever
	data := make([]byte, len(want))
	_, err = io.ReadAtLeast(follow, data, len(want))
	if err != nil {
		return err
	}

	// this should block forever
	errc := make(chan error, 1)
	go func() {
		n, err := follow.Read(make([]byte, 1))
		t.Logf("read %d bytes after closing", n)
		errc <- err
	}()

	if err := follow.Close(); err != nil {
		t.Errorf("failed to close follower: %v", err)
	}

	got := string(data)
	if want != got {
		t.Errorf("want %v, got %v", want, got)
	}

	err = <-errc
	if err != io.EOF {
		t.Errorf("expected EOF after closing follower, got %v", err)
	}

	return nil
}

func TestCanFollowFileFromEnd(t *testing.T) { withTempFile(t, canFollowFileFromEnd) }

func canFollowFileFromEnd(t *testing.T, filename string, file *os.File) error {

	_, err := file.WriteString("shouldn't read this part")
	if err != nil {
		return err
	}

	toWrite := []string{
		"hello,",
		" world!",
	}

	want := strings.Join(toWrite, "")

	follow, err := tailf.Follow(filename, false)
	if err != nil {
		return fmt.Errorf("creating follower: %v", err)
	}

	go func() {
		for _, str := range toWrite {
			log.Printf("writing")
			_, err := file.WriteString(str)
			if err != nil {
				t.Fatalf("failed to write to test file: %v", err)
			}
		}
	}()

	// this should work, without blocking forever
	data := make([]byte, len(want))
	_, err = io.ReadAtLeast(follow, data, len(want))
	if err != nil {
		return err
	}

	// this should block forever
	errc := make(chan error, 1)
	go func() {
		n, err := io.ReadAtLeast(follow, make([]byte, 1), 1)
		t.Logf("read %d bytes after closing", n)
		errc <- err
	}()

	if err := follow.Close(); err != nil {
		t.Errorf("failed to close follower: %v", err)
	}

	got := string(data)
	if want != got {
		t.Errorf("want %v, got %v", want, got)
	}

	err = <-errc
	if err != io.EOF {
		t.Errorf("expected EOF after closing follower, got %v", err)
	}

	return nil
}

func withTempFile(t *testing.T, action func(t *testing.T, filename string, file *os.File) error) {
	timeout := time.AfterFunc(time.Second*5, func() { panic("too long") })
	defer timeout.Stop()

	dir, err := ioutil.TempDir(os.TempDir(), "tailf_test_dir")
	if err != nil {
		t.Fatalf("couldn't create temp dir: %v", err)
	}
	file, err := ioutil.TempFile(dir, "tailf_test")
	if err != nil {
		t.Fatalf("couldn't create temp file: %v", err)
	}
	defer os.RemoveAll(dir)
	defer file.Close()

	err = action(t, file.Name(), file)
	if err != nil {
		t.Errorf("failure: %v", err)
	}
}
