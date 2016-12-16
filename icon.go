package goversioninfo

import (
	"encoding/binary"
	"io"
	"os"

	"github.com/rakslice/rsrc/coff"
	"github.com/rakslice/rsrc/ico"
	"fmt"
	"syscall"
	"log"
)

// *****************************************************************************
/*
Code from https://github.com/akavel/rsrc

The MIT License (MIT)

Copyright (c) 2013-2014 The rsrc Authors.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
// *****************************************************************************

const (
	rtIcon      = coff.RT_ICON
	rtGroupIcon = coff.RT_GROUP_ICON
	rtManifest  = coff.RT_MANIFEST
)

// on storing icons, see: http://blogs.msdn.com/b/oldnewthing/archive/2012/07/20/10331787.aspx
type gRPICONDIR struct {
	ico.ICONDIR
	Entries []gRPICONDIRENTRY
}

func (group gRPICONDIR) Size() int64 {
	return int64(binary.Size(group.ICONDIR) + len(group.Entries)*binary.Size(group.Entries[0]))
}

type gRPICONDIRENTRY struct {
	ico.IconDirEntryCommon
	ID uint16
}

func addStringGetIndex(curCoff *coff.Coff, rawString string) int {
	// add string entry
	newStringIndex := len(curCoff.DirStrings)

	utf16s, err := syscall.UTF16FromString(rawString)
	if err != nil {
		log.Fatal(fmt.Sprintf("Error converting resource name string '%s': %s", rawString, err))
	}

	nonZeroRunesLen := len(utf16s) - 1

	curCoff.DirStrings = append(curCoff.DirStrings, coff.DirString{uint16(nonZeroRunesLen), utf16s})

	return newStringIndex
}

func addNamedIcon(curCoff *coff.Coff, fname string, resourceName string, newID <-chan uint16) error {
	f, err := os.Open(fname)
	if err != nil {
		return err
	}
	//defer f.Close() don't defer, files will be closed by OS when app closes

	icons, err := ico.DecodeHeaders(f)
	if err != nil {
		return err
	}

	if len(icons) > 0 {
		// RT_ICONs
		group := gRPICONDIR{ICONDIR: ico.ICONDIR{
			Reserved: 0, // magic num.
			Type:     1, // magic num.
			Count:    uint16(len(icons)),
		}}
		for _, icon := range icons {
			id := <-newID
			r := io.NewSectionReader(f, int64(icon.ImageOffset), int64(icon.BytesInRes))
			curCoff.AddResource(rtIcon, id, r)
			group.Entries = append(group.Entries, gRPICONDIRENTRY{IconDirEntryCommon: icon.IconDirEntryCommon, ID: id})
		}

		fmt.Fprintf(os.Stderr, "icon group is %v\n", resourceName)
		newStringIndex := addStringGetIndex(curCoff, resourceName)

		// set id to string index
		idOrName := uint32(newStringIndex) | coff.MASK_NAME

		curCoff.AddResourceIdOrNameRef(rtGroupIcon, idOrName, group)
	}

	return nil
}

func addIcon(coff *coff.Coff, fname string, newID <-chan uint16) error {
	f, err := os.Open(fname)
	if err != nil {
		return err
	}
	//defer f.Close() don't defer, files will be closed by OS when app closes

	icons, err := ico.DecodeHeaders(f)
	if err != nil {
		return err
	}

	if len(icons) > 0 {
		// RT_ICONs
		group := gRPICONDIR{ICONDIR: ico.ICONDIR{
			Reserved: 0, // magic num.
			Type:     1, // magic num.
			Count:    uint16(len(icons)),
		}}
		for _, icon := range icons {
			id := <-newID
			r := io.NewSectionReader(f, int64(icon.ImageOffset), int64(icon.BytesInRes))
			coff.AddResource(rtIcon, id, r)
			group.Entries = append(group.Entries, gRPICONDIRENTRY{IconDirEntryCommon: icon.IconDirEntryCommon, ID: id})
		}
		id := <-newID
		coff.AddResource(rtGroupIcon, id, group)
	}

	return nil
}
