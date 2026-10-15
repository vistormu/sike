package model

// Faces are the five moods, 1 (😞) to 5 (😄): Faces[mood-1]. 0 is no mood.
var Faces = []string{"😞", "🙁", "😐", "🙂", "😄"}

func IsMood(m int) bool { return m >= 1 && m <= len(Faces) }
