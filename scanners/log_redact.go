package scanners

import (
	"git.commsnet.org/commstech/repository-detective/internal/security"
	"github.com/sirupsen/logrus"
)

// logResultInfo logs scanner completion without echoing raw stderr that may contain secrets.
func logResultInfo(logger *logrus.Logger, name string, status Status, findingCount int, detail string) {
	if logger == nil {
		return
	}
	logger.Infof("[SCANNER:%s] status=%s findings=%d detail=%q",
		name, status, findingCount, security.RedactLogField(detail, 120))
}
