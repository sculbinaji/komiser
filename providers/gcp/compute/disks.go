package compute

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tailwarden/komiser/models"
	"github.com/tailwarden/komiser/providers"
	"github.com/tailwarden/komiser/utils"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	compute "cloud.google.com/go/compute/apiv1"
	computepb "cloud.google.com/go/compute/apiv1/computepb"
)

func Disks(ctx context.Context, client providers.ProviderClient) ([]models.Resource, error) {
	resources := make([]models.Resource, 0)

	disksClient, err := compute.NewDisksRESTClient(ctx, option.WithCredentials(client.GCPClient.Credentials))
	if err != nil {
		logrus.WithError(err).Errorf("failed to create compute client")
		return resources, err
	}

	req := &computepb.AggregatedListDisksRequest{
		Project: client.GCPClient.Credentials.ProjectID,
	}
	disks := disksClient.AggregatedList(ctx, req)

	for {
		disksListPair, err := disks.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			logrus.WithError(err).Errorf("failed to list instances")
			return resources, err
		}
		if len(disksListPair.Value.Disks) == 0 {
			continue
		}

		for _, disk := range disksListPair.Value.Disks {
			tags := make([]models.Tag, 0)
			if disk.Labels != nil {
				for key, value := range disk.Labels {
					tags = append(tags, models.Tag{
						Key:   key,
						Value: value,
					})
				}
			}

			zone := utils.GcpExtractZoneFromURL(disk.GetZone())

			resources = append(resources, models.Resource{
				Provider:   "GCP",
				Account:    client.Name,
				Service:    "Compute Disk",
				ResourceId: fmt.Sprintf("%d", disk.GetId()),
				Region:     zone,
				Name:       disk.GetName(),
				FetchedAt:  time.Now(),
				Tags:       tags,
				Link:       fmt.Sprintf("https://console.cloud.google.com/compute/disksDetail/zones/%s/disks/%s?project=%s", zone, disk.GetName(), client.GCPClient.Credentials.ProjectID),
			})
		}
	}

	logrus.WithFields(logrus.Fields{
		"provider":  "GCP",
		"account":   client.Name,
		"service":   "Compute Engine",
		"resources": len(resources),
	}).Info("Fetched resources")

	return resources, nil
}
