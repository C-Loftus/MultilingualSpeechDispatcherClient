package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	speechd "github.com/ilyapashuk/go-speechd"

	"github.com/charmbracelet/log"
	"github.com/pemistahl/lingua-go"
)

// Create a mapping from string to lingua.Language
var MapStringToEnumLanguage = func() map[string]lingua.Language {
	m := make(map[string]lingua.Language)
	for i := lingua.Afrikaans; i <= lingua.Unknown; i++ {
		m[i.String()] = i
	}
	return m
}()

// sometimes we fail to connect but can succeed on a retry
// see https://github.com/ilyapashuk/go-speechd/issues/1
func openSpeechClient(retries int, delay time.Duration) (*speechd.SpeechdSession, error) {
	var client *speechd.SpeechdSession
	var err error
	for i := range retries {
		client, err = speechd.Open()
		if err == nil {
			log.Info("Connected to Speech Dispatcher")
			return client, nil
		}
		log.Warnf("Failed to connect to Speech Dispatcher (attempt %d/%d): %v", i+1, retries, err)
		time.Sleep(delay)
	}
	return nil, fmt.Errorf("failed to connect to Speech Dispatcher after %d attempts", retries)
}

// handle command line flags
func parseFlags() (StringSlice, error) {
	var languages StringSlice
	flag.Var(&languages, "use-languages", "A list of languages used for language detection. Specify as a comma-separated list (i.e. English,Spanish,French)")

	listLanguages := flag.Bool("list-languages", false, "Print out supported languages and exit")

	flag.Parse()
	if *listLanguages {
		for language := range MapStringToEnumLanguage {
			fmt.Println(language)
		}
		os.Exit(0)
	}

	if len(languages) == 0 {
		return nil, fmt.Errorf("must specify at least one language")
	}

	return languages, nil
}

func scanAndSpeak(input io.Reader, client *speechd.SpeechdSession, detector lingua.LanguageDetector) error {

	fmt.Println("Enter text to detect language (CTRL+D to exit):")
	scanner := bufio.NewScanner(input)
	// we need to enable event notifications to be able to wait for spoken messages to complete
	if err := client.SetEventNotifications(true); err != nil {
		return err
	}

	for scanner.Scan() {
		scanned_text := scanner.Text()

		for _, result := range detector.DetectMultipleLanguagesOf(scanned_text) {
			subset := scanned_text[result.StartIndex():result.EndIndex()]
			iso_lang := result.Language().IsoCode639_1().String()
			log.Debugf("Detected language %s for substring '%s'", iso_lang, subset)
			if err := client.SetLanguage(iso_lang); err != nil {
				return err
			}
			msg, err := client.Speak(subset)
			if err != nil {
				return err
			}
			msg.Wait()
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stdin: %v", err)
	}
	return nil
}

func main() {

	log.SetLevel(log.DebugLevel)

	languages, err := parseFlags()
	log.Debugf("Trying to detect the following languages: %v", languages)
	if err != nil {
		log.Fatal(err)
	}

	// map the languages passed in to an enum we can use in lingua
	var linguaLanguages []lingua.Language
	for _, language := range strings.Split(languages.String(), ",") {
		linguaLanguage, ok := MapStringToEnumLanguage[language]
		if !ok {
			log.Fatalf("unknown language: %s", language)
		}
		linguaLanguages = append(linguaLanguages, linguaLanguage)
	}
	if len(linguaLanguages) < 2 {
		log.Fatal("at least two languages must be specified")
	}

	detector := lingua.NewLanguageDetectorBuilder().
		FromLanguages(linguaLanguages...).
		Build()

	client, err := openSpeechClient(5, 1*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	if err := client.SetClientName("bilingual-voice-test", "bilingual-voice-test", "bilingual-voice-test"); err != nil {
		log.Fatal(err)
	}
	if err := client.SetOutputModule("espeak-ng"); err != nil {
		log.Fatal(err)
	}

	// make sure that we clean up the client on exit,
	// even if we panic or send a signal to terminate
	defer client.Close()
	defer func() {
		if r := recover(); r != nil {
			client.Close()
		}
	}()
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		client.Close()
		os.Exit(0)
	}()

	if err := scanAndSpeak(os.Stdin, client, detector); err != nil {
		log.Fatal(err)
	}
}
