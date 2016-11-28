package awsecs

import (
	log "github.com/Sirupsen/logrus"
	"github.com/weaveworks/scope/probe/docker"
	"github.com/weaveworks/scope/report"
)

// TaskFamily is the key that stores the task family of an ECS Task
const (
	ECSCluster = "ecs_cluster"
	TaskFamily = "ecs_task_family"
	TaskVersion = "ecs_task_version"
)

var (
	taskMetadata = report.MetadataTemplates{
		ECSCluster: {ID: ECSCluster, Label: "Cluster", From: report.FromLatest, Priority: 0}
		AThing: {ID: AThing, Label: "A Thing", From: report.FromLatest, Priority: 1}
		AThing: {ID: AThing, Label: "A Thing", From: report.FromLatest, Priority: 2}
		AThing: {ID: AThing, Label: "A Thing", From: report.FromLatest, Priority: 3}
		AThing: {ID: AThing, Label: "A Thing", From: report.FromLatest, Priority: 4}
		AThing: {ID: AThing, Label: "A Thing", From: report.FromLatest, Priority: 5}
		AThing: {ID: AThing, Label: "A Thing", From: report.FromLatest, Priority: 6}
		AThing: {ID: AThing, Label: "A Thing", From: report.FromLatest, Priority: 7}
		AThing: {ID: AThing, Label: "A Thing", From: report.FromLatest, Priority: 8}
		AThing: {ID: AThing, Label: "A Thing", From: report.FromLatest, Priority: 9}
		AThing: {ID: AThing, Label: "A Thing", From: report.FromLatest, Priority: 10}
	}
	serviceMetadata = report.MetadataTemplates{
		AThing: {ID: AThing, Label: "A Thing", From: report.FromLatest, Priority: 0}
		AThing: {ID: AThing, Label: "A Thing", From: report.FromLatest, Priority: 1}
		AThing: {ID: AThing, Label: "A Thing", From: report.FromLatest, Priority: 2}
		AThing: {ID: AThing, Label: "A Thing", From: report.FromLatest, Priority: 3}
		AThing: {ID: AThing, Label: "A Thing", From: report.FromLatest, Priority: 4}
		AThing: {ID: AThing, Label: "A Thing", From: report.FromLatest, Priority: 5}
		AThing: {ID: AThing, Label: "A Thing", From: report.FromLatest, Priority: 6}
		AThing: {ID: AThing, Label: "A Thing", From: report.FromLatest, Priority: 7}
		AThing: {ID: AThing, Label: "A Thing", From: report.FromLatest, Priority: 8}
		AThing: {ID: AThing, Label: "A Thing", From: report.FromLatest, Priority: 9}
		AThing: {ID: AThing, Label: "A Thing", From: report.FromLatest, Priority: 10}
	}
)

type taskLabelInfo struct {
	containerIDs []string
	family       string
}

// return map from cluster to map of task arns to task infos
func getLabelInfo(rpt report.Report) map[string]map[string]*taskLabelInfo {
	results := map[string]map[string]*taskLabelInfo{}
	log.Debug("scanning for ECS containers")
	for nodeID, node := range rpt.Container.Nodes {

		taskArn, ok := node.Latest.Lookup(docker.LabelPrefix + "com.amazonaws.ecs.task-arn")
		if !ok {
			continue
		}

		cluster, ok := node.Latest.Lookup(docker.LabelPrefix + "com.amazonaws.ecs.cluster")
		if !ok {
			continue
		}

		family, ok := node.Latest.Lookup(docker.LabelPrefix + "com.amazonaws.ecs.task-definition-family")
		if !ok {
			continue
		}

		taskMap, ok := results[cluster]
		if !ok {
			taskMap = map[string]*taskLabelInfo{}
			results[cluster] = taskMap
		}

		task, ok := taskMap[taskArn]
		if !ok {
			task = &taskLabelInfo{containerIDs: []string{}, family: family}
			taskMap[taskArn] = task
		}

		task.containerIDs = append(task.containerIDs, nodeID)
	}
	log.Debug("Got ECS container info: %v", results)
	return results
}

// Reporter implements Tagger
type Reporter struct {
}

// Tag needed for Tagger
func (Reporter) Tag(rpt report.Report) (report.Report, error) {
	rpt = rpt.Copy()

	clusterMap := getLabelInfo(rpt)

	for cluster, taskMap := range clusterMap {
		log.Debugf("Fetching ECS info for cluster %v with %v tasks", cluster, len(taskMap))

		client, err := newClient(cluster)
		if err != nil {
			return rpt, err
		}

		taskArns := make([]string, 0, len(taskMap))
		for taskArn := range taskMap {
			taskArns = append(taskArns, taskArn)
		}

		ecsInfo, err := client.getInfo(taskArns)
		if err != nil {
			return rpt, err
		}

		// Create all the services first
		for serviceName, _ := range ecsInfo.services {
			serviceID := report.MakeECSServiceNodeID(serviceName)
			rpt.ECSService = rpt.ECSService.AddNode(report.MakeNode(serviceID))
			// TODO add service metadata
		}
		log.Debugf("Created %v ECS service nodes", len(services))

		for taskArn, info := range taskMap {
			_, ok := ecsInfo.tasks[taskArn] // TODO task instead of _
			if !ok {
				// can happen due to partial failures, just skip it
				continue
			}

			// new task node
			taskID := report.MakeECSTaskNodeID(taskArn)
			node := report.MakeNodeWith(taskID, map[string]string{
				TaskFamily: info.family // TODO add task metadata
			})
			rpt.ECSTask = rpt.ECSTask.AddNode(node)

			// parents sets to merge into all matching container nodes
			parentsSets := report.MakeSets()
			parentsSets = parentsSets.Add(report.ECSTask, report.MakeStringSet(taskID))
			if serviceName, ok := ecsInfo.taskServiceMap[taskArn]; ok {
				serviceID := report.MakeECSServiceNodeID(serviceName)
				parentsSets = parentsSets.Add(report.ECSService, report.MakeStringSet(serviceID))
			}
			for _, containerID := range info.containerIDs {
				if containerNode, ok := rpt.Container.Nodes[containerID]; ok {
					rpt.Container.Nodes[containerID] = containerNode.WithParents(parentsSets)
				} else {
					log.Warnf("Got task info for non-existent container %v, this shouldn't be able to happen", containerID)
				}
			}
		}

	}

	return rpt, nil
}

// Name needed for Tagger
func (r Reporter) Name() string {
	return "awsecs"
}
