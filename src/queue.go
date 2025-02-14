package tango

func (s *server) removeJobFromQueue(jobID string) {
	index := -1
	for i, id := range s.jobQueue {
		if id == jobID {
			index = i
			break
		}
	}
	if index != -1 {
		s.jobQueue = append(s.jobQueue[:index], s.jobQueue[index+1:]...)
	}
}
