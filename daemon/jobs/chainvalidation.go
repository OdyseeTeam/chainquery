package jobs

import (
	"fmt"
	"time"

	"github.com/lbryio/chainquery/apiactions"
	"github.com/lbryio/chainquery/model"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

var validatingChain = false

const chainValidationJob = "chainvalidationjob"

func ValidateChain() {
	if !validatingChain {
		go func() {
			var job *model.JobStatus
			exists, err := model.JobStatuses(qm.Where(model.JobStatusColumns.JobName+"=?", chainValidationJob)).ExistsG()
			if err != nil {
				logrus.Error("Chain Validation: ", err)
				return
			}
			if !exists {
				job = &model.JobStatus{JobName: chainValidationJob}
			} else {
				job, err = model.JobStatuses(qm.Where(model.JobStatusColumns.JobName+"=?", chainValidationJob)).OneG()
				if err != nil {
					logrus.Error("Chain Validation: ", err)
					return
				}
			}
			startOfChain := uint64(0)
			missingData, err := apiactions.ValidateChain(&startOfChain, nil)
			if err != nil {
				job.ErrorMessage.SetValid(err.Error())
				job.IsSuccess = false
			}

			if len(missingData) > 0 {
				job.ErrorMessage.SetValid(fmt.Sprintf("%d pieces of missing data"))
			}

			job.LastSync = time.Now()

			err = job.UpsertG(boil.Infer(), boil.Infer())
			if err != nil {
				logrus.Error("Chain Validation: ", err)
				return
			}

			return

		}()
	}
}
