package m3u

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Playlist is a type that represents an m3u playlist containing 0 or more tracks
type Playlist struct {
	Tracks []Track
}

// A Tag is a simple key/value pair
type Tag struct {
	Name  string
	Value string
}

// Track represents an m3u track with a Name, Lengh, URI and a set of tags
type Track struct {
	Name   string
	Length int
	URI    string
	Tags   []Tag
	Group  string
}

// Parse parses an m3u playlist with the given file name and returns a Playlist
func Parse(fileName string) (playlist Playlist, err error) {
	var f io.ReadCloser
	var data *http.Response
	if strings.HasPrefix(fileName, "http://") || strings.HasPrefix(fileName, "https://") {
		data, err = http.Get(fileName)
		f = data.Body
	} else {
		f, err = os.Open(fileName)
	}

	if err != nil {
		err = errors.New("Unable to open playlist file")
		return
	}
	defer f.Close()

	onFirstLine := true
	scanner := bufio.NewScanner(f)
	tagsRegExp, _ := regexp.Compile("([a-zA-Z0-9-]+?)=\"([^\"]+)\"")

	for scanner.Scan() {
		line := scanner.Text()
		if onFirstLine && !strings.HasPrefix(line, "#EXTM3U") {
			err = errors.New("Invalid m3u file format. Expected #EXTM3U file header")
			return
		}

		onFirstLine = false

		if strings.HasPrefix(line, "#EXTINF") {
			line := strings.Replace(line, "#EXTINF:", "", -1)
			trackInfo := strings.Split(line, ",")
			if len(trackInfo) < 2 {
				err = errors.New("Invalid m3u file format. Expected EXTINF metadata to contain track length and name data")
				return
			}
			length, parseErr := strconv.Atoi(strings.Split(strings.TrimSpace(trackInfo[0]), " ")[0])
			if parseErr != nil {
				err = errors.New("Unable to parse length")
				return
			}
			track := &Track{strings.Trim(trackInfo[1], " "), length, "", nil, ""}
			tagList := tagsRegExp.FindAllString(line, -1)
			for i := range tagList {
				tagInfo := strings.Split(tagList[i], "=")
				tag := &Tag{tagInfo[0], strings.Replace(tagInfo[1], "\"", "", -1)}
				track.Tags = append(track.Tags, *tag)
			}
			playlist.Tracks = append(playlist.Tracks, *track)
		} else if strings.HasPrefix(line, "#EXTGRP") {
			line := strings.Replace(line, "#EXTGRP:", "", -1)
			playlist.Tracks[len(playlist.Tracks)-1].Group = line
		} else if strings.HasPrefix(line, "#") || line == "" {
			continue
		} else if len(playlist.Tracks) == 0 {
			err = errors.New("URI provided for playlist with no tracks")
			return
		} else {
			playlist.Tracks[len(playlist.Tracks)-1].URI = strings.Trim(line, " ")
		}
	}

	return playlist, nil
}

// Marshall Playlist to an m3u file.
func Marshall(p Playlist) (io.Reader, error) {
	buf := new(bytes.Buffer)
	w := bufio.NewWriter(buf)
	if err := MarshallInto(p, w,true); err != nil {
		return nil, err
	}

	return buf, nil
}

// MarshallInto a *bufio.Writer a Playlist.
func MarshallInto(p Playlist, into *bufio.Writer, writeHeader bool) error {
	if writeHeader {
		into.WriteString("#EXTM3U\n")
	}
	for _, track := range p.Tracks {
		into.WriteString("#EXTINF:")
		into.WriteString(fmt.Sprintf("%d ", track.Length))
		for i := range track.Tags {
			if i == len(track.Tags)-1 {
				into.WriteString(fmt.Sprintf("%s=%q", track.Tags[i].Name, track.Tags[i].Value))
				continue
			}
			into.WriteString(fmt.Sprintf("%s=%q ", track.Tags[i].Name, track.Tags[i].Value))
		}
		into.WriteString(", ")
		if track.Group != "" {
			into.WriteString(fmt.Sprintf("%s\n#EXTGRP:%s\n%s\n", track.Name, track.Group, track.URI))
		}else{
			into.WriteString(fmt.Sprintf("%s\n%s\n", track.Name, track.URI))
		}
	}

	return into.Flush()
}
