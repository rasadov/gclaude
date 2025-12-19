package notify

import (
	"os"
	"os/exec"
	"path/filepath"
)

var defaultSoundPaths = []string{
	"/usr/share/sounds/freedesktop/stereo/complete.oga",
	"/usr/share/sounds/freedesktop/stereo/message.oga",
	"/usr/share/sounds/freedesktop/stereo/bell.oga",
	"/usr/share/sounds/gnome/default/alerts/drip.ogg",
	"/usr/share/sounds/ubuntu/stereo/message.ogg",
}

func Sound(customPath string) error {
	soundFile := customPath
	if soundFile == "" {
		soundFile = findDefaultSound()
	}

	if soundFile == "" {
		return playBeep()
	}

	return playFile(soundFile)
}

func findDefaultSound() string {
	for _, path := range defaultSoundPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func playFile(path string) error {
	ext := filepath.Ext(path)

	switch ext {
	case ".ogg", ".oga":
		if _, err := exec.LookPath("paplay"); err == nil {
			return exec.Command("paplay", path).Run()
		}
		if _, err := exec.LookPath("ogg123"); err == nil {
			return exec.Command("ogg123", "-q", path).Run()
		}
	case ".wav":
		if _, err := exec.LookPath("aplay"); err == nil {
			return exec.Command("aplay", "-q", path).Run()
		}
		if _, err := exec.LookPath("paplay"); err == nil {
			return exec.Command("paplay", path).Run()
		}
	case ".mp3":
		if _, err := exec.LookPath("mpg123"); err == nil {
			return exec.Command("mpg123", "-q", path).Run()
		}
	}

	if _, err := exec.LookPath("paplay"); err == nil {
		return exec.Command("paplay", path).Run()
	}
	if _, err := exec.LookPath("aplay"); err == nil {
		return exec.Command("aplay", "-q", path).Run()
	}

	return playBeep()
}

func playBeep() error {
	if _, err := exec.LookPath("paplay"); err == nil {
		for _, path := range defaultSoundPaths {
			if _, err := os.Stat(path); err == nil {
				return exec.Command("paplay", path).Run()
			}
		}
	}

	return exec.Command("printf", "\a").Run()
}
