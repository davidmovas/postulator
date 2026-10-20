package pagemap

func StatusFromWordPress(wordpress string) Status {
	if wordpress == "publish" {
		return StatusPublished
	}
	return StatusExists
}
