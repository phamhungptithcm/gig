package conflict

import (
	"bytes"
	"fmt"
)

var (
	markerCurrent  = []byte("<<<<<<<")
	markerBase     = []byte("|||||||")
	markerDivider  = []byte("=======")
	markerIncoming = []byte(">>>>>>>")
)

func ParseFile(content []byte) (ParsedFile, error) {
	return parseFile(content), nil
}

func parseFile(content []byte) ParsedFile {
	lines := splitLines(content)
	segments := make([]Segment, 0, len(lines))
	blocks := make([]Block, 0, 4)
	var plain bytes.Buffer

	lineNumber := 1
	for i := 0; i < len(lines); {
		line := lines[i]
		if !hasMarker(line, markerCurrent) {
			plain.Write(line)
			lineNumber++
			i++
			continue
		}

		if plain.Len() > 0 {
			segments = append(segments, Segment{Text: bytes.Clone(plain.Bytes())})
			plain.Reset()
		}

		startLine := lineNumber
		i++
		lineNumber++

		currentStart := i
		baseStart := -1
		divider := -1
		incomingStart := -1
		for ; i < len(lines); i++ {
			switch {
			case hasMarker(lines[i], markerBase) && baseStart == -1 && divider == -1:
				baseStart = i
			case hasMarker(lines[i], markerDivider) && divider == -1:
				divider = i
				incomingStart = i + 1
			case hasMarker(lines[i], markerIncoming) && divider != -1:
				endLine := lineNumber
				currentEnd := divider
				if baseStart != -1 {
					currentEnd = baseStart
				}

				baseEnd := divider
				if baseStart == -1 {
					baseEnd = -1
				}

				block := Block{
					Index:     len(blocks),
					StartLine: startLine,
					EndLine:   endLine,
					Current:   joinLines(lines[currentStart:currentEnd]),
					Incoming:  joinLines(lines[incomingStart:i]),
				}
				if baseStart != -1 {
					block.Base = joinLines(lines[baseStart+1 : baseEnd])
				}

				blocks = append(blocks, block)
				segments = append(segments, Segment{Block: &Block{Index: block.Index}})

				i++
				lineNumber++
				goto nextBlock
			}
			lineNumber++
		}

		plain.Write(markerCurrent)
		plain.WriteByte('\n')
		break

	nextBlock:
	}

	if plain.Len() > 0 {
		segments = append(segments, Segment{Text: bytes.Clone(plain.Bytes())})
	}

	parsed := ParsedFile{
		Segments: segments,
		Blocks:   blocks,
	}
	for i := range parsed.Segments {
		if parsed.Segments[i].Block == nil {
			continue
		}
		index := parsed.Segments[i].Block.Index
		if index < 0 || index >= len(parsed.Blocks) {
			parsed.Segments[i].Block = nil
			continue
		}
		parsed.Segments[i].Block = &parsed.Blocks[index]
	}
	return parsed
}

func ApplyResolution(parsed ParsedFile, blockIndex int, choice ResolutionChoice) ([]byte, error) {
	if blockIndex < 0 || blockIndex >= len(parsed.Blocks) {
		return nil, fmt.Errorf("conflict block %d not found", blockIndex)
	}

	var out bytes.Buffer
	for _, segment := range parsed.Segments {
		if segment.Block == nil {
			out.Write(segment.Text)
			continue
		}

		if segment.Block.Index != blockIndex {
			writeBlock(&out, *segment.Block)
			continue
		}

		switch choice {
		case ResolutionCurrent:
			out.Write(segment.Block.Current)
		case ResolutionIncoming:
			out.Write(segment.Block.Incoming)
		case ResolutionBothCurrentFirst:
			out.Write(segment.Block.Current)
			out.Write(segment.Block.Incoming)
		case ResolutionBothIncomingFirst:
			out.Write(segment.Block.Incoming)
			out.Write(segment.Block.Current)
		default:
			return nil, fmt.Errorf("unsupported resolution choice %q", choice)
		}
	}

	return out.Bytes(), nil
}

func writeBlock(out *bytes.Buffer, block Block) {
	out.WriteString("<<<<<<< ")
	out.WriteString(block.CurrentRef.Label)
	out.WriteByte('\n')
	out.Write(block.Current)
	if len(block.Base) > 0 {
		out.WriteString("||||||| base\n")
		out.Write(block.Base)
	}
	out.WriteString("=======\n")
	out.Write(block.Incoming)
	out.WriteString(">>>>>>> ")
	out.WriteString(block.IncomingRef.Label)
	out.WriteByte('\n')
}

func splitLines(content []byte) [][]byte {
	if len(content) == 0 {
		return nil
	}

	lines := make([][]byte, 0, bytes.Count(content, []byte{'\n'})+1)
	start := 0
	for i, b := range content {
		if b != '\n' {
			continue
		}
		lines = append(lines, bytes.Clone(content[start:i+1]))
		start = i + 1
	}
	if start < len(content) {
		lines = append(lines, bytes.Clone(content[start:]))
	}
	return lines
}

func joinLines(lines [][]byte) []byte {
	if len(lines) == 0 {
		return nil
	}

	var out bytes.Buffer
	for _, line := range lines {
		out.Write(line)
	}
	return out.Bytes()
}

func hasMarker(line, prefix []byte) bool {
	return bytes.HasPrefix(line, prefix)
}

func ContainsConflictMarkers(content []byte) bool {
	for _, line := range splitLines(content) {
		if hasMarker(line, markerCurrent) || hasMarker(line, markerDivider) || hasMarker(line, markerIncoming) {
			return true
		}
	}
	return false
}
