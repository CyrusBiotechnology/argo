package commands

import (
	"fmt"
	wfv1 "github.com/cyrusbiotechnology/argo/pkg/apis/workflow/v1alpha1"
	"github.com/cyrusbiotechnology/argo/workflow/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"sort"
	"text/tabwriter"
	"time"
)

type TemplateDuration struct {
	Name     string
	Duration time.Duration
}

func NewCostCommand() *cobra.Command {
	var costPerHour float64
	var command = &cobra.Command{
		Use:   "cost WORKFLOW",
		Short: "display a cost estimate for a workflow",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			wfClient := InitWorkflowClient()
			for _, arg := range args {
				wf, err := wfClient.Get(arg, metav1.GetOptions{})
				if err != nil {
					log.Fatal(err)
				}
				err = util.DecompressWorkflow(wf)
				if err != nil {
					log.Fatal(err)
				}
				templateRuntimes := getTemplateRuntimes(wf)
				const fmtStr = "%-20s %v\n"

				fmt.Printf("Assuming a total cost of $%f per CPU Hour and fully utilized nodes\n\n", costPerHour)

				w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
				fmt.Fprint(w, "TEMPLATE\tDURATION\tCOST\n")

				totalDuration := time.Duration(0)
				for _, templateDuration := range templateRuntimes {
					totalDuration += templateDuration.Duration
					fmt.Fprintf(w,
						"%s\t%v\t$%f\n",
						templateDuration.Name,
						templateDuration.Duration,
						templateDuration.Duration.Hours()*costPerHour)
				}
				_ = w.Flush()
				fmt.Print("\n")
				fmt.Printf("%-15s %v\n", "Total CPU time:", totalDuration)
				fmt.Printf("%-15s $%f\n", "Total cost:", totalDuration.Hours()*costPerHour)
			}

		},
	}

	command.Flags().Float64Var(&costPerHour, "cost", 0.01, "Cost per hour in dollars (Default $0.01)")

	return command
}

func getTemplateRuntimes(wf *wfv1.Workflow) []TemplateDuration {

	perTemplateTotals := map[string]time.Duration{}

	for _, node := range wf.Status.Nodes {
		if node.Type != wfv1.NodeTypePod {
			continue
		}

		if node.Phase != wfv1.NodeSucceeded {
			continue
		}

		startTime := node.StartedAt
		endTime := node.FinishedAt

		duration := endTime.Sub(startTime.Time)
		if _, ok := perTemplateTotals[node.TemplateName]; ok {
			perTemplateTotals[node.TemplateName] = perTemplateTotals[node.TemplateName] + duration
		} else {
			perTemplateTotals[node.TemplateName] = duration
		}

	}

	var templateDurations []TemplateDuration

	for templateName, duration := range perTemplateTotals {
		templateDurations = append(templateDurations, TemplateDuration{
			Name:     templateName,
			Duration: duration,
		})
	}

	sort.Slice(templateDurations, func(i, j int) bool {
		return templateDurations[i].Duration > templateDurations[j].Duration
	})

	return templateDurations
}
