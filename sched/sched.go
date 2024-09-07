package sched

import (
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/perlogix/pal/config"
	"github.com/perlogix/pal/data"
	"github.com/perlogix/pal/utils"
)

var (
	Jobs *[]gocron.Job
)

func cronTask(res data.ResourceData) string {
	cmdOutput, err := utils.CmdRun(res.Cmd)
	if err != nil {
		return err.Error()
	}

	fmt.Printf("%s\n", fmt.Sprintf(`{"time":"%s","target":"%s","job_success":"%t"}`, time.Now().Format(time.RFC3339), res.Target, true))

	return cmdOutput
}

func CronStart(r map[string][]data.ResourceData) error {
	var sched gocron.Scheduler

	loc, err := time.LoadLocation(config.GetConfigStr("http_schedule_tz"))
	if err != nil {
		return err
	}

	sched, err = gocron.NewScheduler(gocron.WithLocation(loc))
	if err != nil {
		return err
	}

	for k, v := range r {
		for _, e := range v {
			if e.Schedule != "" {
				_, err := sched.NewJob(
					gocron.CronJob(e.Schedule, false),
					gocron.NewTask(cronTask, e),
					gocron.WithName(k+"/"+e.Target),
				)
				if err != nil {
					return err
				}
			}
		}
	}

	sched.Start()
	jobs := sched.Jobs()
	Jobs = &jobs

	return nil

}
