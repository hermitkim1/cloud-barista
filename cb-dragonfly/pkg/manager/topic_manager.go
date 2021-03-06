package manager

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/cloud-barista/cb-dragonfly/pkg/localstore"
	"github.com/cloud-barista/cb-dragonfly/pkg/types"
	"github.com/cloud-barista/cb-dragonfly/pkg/util"
)

type TopicManager struct {
	IdealCollectorGroupMap         map[int][]string
	IdealCollectorPerAgentCntSlice []int
}

var once sync.Once
var topicManager TopicManager

func TopicMangerInit() {
	topicManager.IdealCollectorGroupMap = map[int][]string{}
	topicManager.IdealCollectorPerAgentCntSlice = []int{}
}

func TopicMangerInstance() *TopicManager {
	once.Do(func() {
		TopicMangerInit()
	})
	return &topicManager
}

func (t *TopicManager) SetTopicToCollectorBasedTheNumberOfAgent(topicList []string, maxHostCount int) {
	t.IdealCollectorGroupMap, t.IdealCollectorPerAgentCntSlice = util.MakeCollectorTopicMap(topicList, maxHostCount)
	if len(t.IdealCollectorGroupMap) == 0 && len(t.IdealCollectorPerAgentCntSlice) == 0 {
		_ = localstore.GetInstance().StorePut(fmt.Sprintf("%s/%d", types.COLLECTORGROUPTOPIC, 0), "")
		return
	}
	for collectorIdx, collectorTopics := range t.IdealCollectorGroupMap {
		for i := 0; i < len(collectorTopics); i++ {
			_ = localstore.GetInstance().StorePut(fmt.Sprintf("%s/%s", types.TOPIC, collectorTopics[i]), strconv.Itoa(collectorIdx))
		}
		_ = localstore.GetInstance().StorePut(fmt.Sprintf("%s/%d", types.COLLECTORGROUPTOPIC, collectorIdx), util.MergeTopicsToOneString(collectorTopics))
	}
}

func (t *TopicManager) DeleteAllTopicsInfo() error {
	err := localstore.GetInstance().StoreDelList(fmt.Sprintf("%s/", types.COLLECTORGROUPTOPIC))
	if err != nil {
		return err
	}
	return nil
}

func (t *TopicManager) DeleteTopics(deletedTopicList []string) error {
	if len(deletedTopicList) == 0 {
		return nil
	}
	changedCollectorMapIdx := map[string][]string{}
	for _, topic := range deletedTopicList {
		collectorIdx := localstore.GetInstance().StoreGet(fmt.Sprintf("%s/%s", types.TOPIC, topic))
		topicStrings := localstore.GetInstance().StoreGet(fmt.Sprintf("%s/%s", types.COLLECTORGROUPTOPIC, collectorIdx))
		if _, ok := changedCollectorMapIdx[collectorIdx]; !ok {
			changedCollectorMapIdx[collectorIdx] = util.SplitOneStringToTopicsSlice(topicStrings)
		}
		changedCollectorMapIdx[collectorIdx] = util.ReturnDiffTopicList(changedCollectorMapIdx[collectorIdx], []string{topic})
		err := localstore.GetInstance().StoreDelete(fmt.Sprintf("%s/%s", types.TOPIC, topic))
		if err != nil {
			return err
		}
		idx, _ := strconv.Atoi(collectorIdx)
		if len(t.IdealCollectorPerAgentCntSlice) != 0 {
			t.IdealCollectorPerAgentCntSlice[idx] -= 1
		}
	}
	for key, _ := range changedCollectorMapIdx {
		err := localstore.GetInstance().StorePut(fmt.Sprintf("%s/%s", types.COLLECTORGROUPTOPIC, key), util.MergeTopicsToOneString(changedCollectorMapIdx[key]))
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *TopicManager) AddNewTopics(newTopicList []string, maxHostCount int) error {
	if len(newTopicList) == 0 {
		return nil
	}
	for _, topic := range newTopicList {
		for collectorIdx, collectorTopicCnt := range t.IdealCollectorPerAgentCntSlice {
			if collectorTopicCnt < maxHostCount {
				err := localstore.GetInstance().StorePut(fmt.Sprintf("%s/%s", types.TOPIC, topic), strconv.Itoa(collectorIdx))
				if err != nil {
					return err
				}
				err = localstore.GetInstance().StorePut(fmt.Sprintf("%s/%d", types.COLLECTORGROUPTOPIC, collectorIdx), localstore.GetInstance().StoreGet(fmt.Sprintf("%s/%d", types.COLLECTORGROUPTOPIC, collectorIdx))+"&"+topic)
				if err != nil {
					return err
				}
				t.IdealCollectorPerAgentCntSlice[collectorIdx] += 1
				break
			}
		}
	}
	return nil
}

func (t *TopicManager) SetTopicToCollectorBasedCSPTypeOfAgent(topicList []string) {
	t.IdealCollectorGroupMap = util.MakeCollectorTopicMapBasedCSP(topicList)
	for collectorIdx, collectorTopics := range t.IdealCollectorGroupMap {
		if len(collectorTopics) != 0 {
			for i := 0; i < len(collectorTopics); i++ {
				_ = localstore.GetInstance().StorePut(fmt.Sprintf("%s/%s", types.TOPIC, collectorTopics[i]), strconv.Itoa(collectorIdx))
			}
			_ = localstore.GetInstance().StorePut(fmt.Sprintf("%s/%d", types.COLLECTORGROUPTOPIC, collectorIdx), util.MergeTopicsToOneString(collectorTopics))
		} else {
			_ = localstore.GetInstance().StorePut(fmt.Sprintf("%s/%d", types.COLLECTORGROUPTOPIC, collectorIdx), "")
		}
	}
}

func (t *TopicManager) AddNewTopicsOnCSPCollector(newTopicList []string) error {
	if len(newTopicList) == 0 {
		return nil
	}
	for _, topic := range newTopicList {
		splitTopic := strings.Split(topic, "_")
		cspType := strings.ToUpper(splitTopic[len(splitTopic)-1])
		var collectorIdx int
		switch cspType {
		case types.ALIBABA:
			collectorIdx = 0
			break
		case types.AWS:
			collectorIdx = 1
			break
		case types.AZURE:
			collectorIdx = 2
			break
		case types.CLOUDIT:
			collectorIdx = 3
			break
		case types.CLOUDTWIN:
			collectorIdx = 4
			break
		case types.DOCKER:
			collectorIdx = 5
			break
		case types.GCP:
			collectorIdx = 6
			break
		case types.OPENSTACK:
			collectorIdx = 7
			break
		}
		err := localstore.GetInstance().StorePut(fmt.Sprintf("%s/%s", types.TOPIC, topic), strconv.Itoa(collectorIdx))
		if err != nil {
			return err
		}
		err = localstore.GetInstance().StorePut(fmt.Sprintf("%s/%d", types.COLLECTORGROUPTOPIC, collectorIdx), localstore.GetInstance().StoreGet(fmt.Sprintf("%s/%d", types.COLLECTORGROUPTOPIC, collectorIdx))+"&"+topic)
		if err != nil {
			return err
		}
	}
	return nil
}
