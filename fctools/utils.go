package fctools

import "time"

func FormatTime(timestamp uint32) string {
	return time.Unix(int64(timestamp)+FARCASTER_EPOCH, 0).Format("2006-01-02 15:04:05")
}
