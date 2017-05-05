package main

import (
	"encoding/json"
	"fmt"
)

const (
	videoUUIDField     = "id"
	relatedField       = "related"
	deletedField       = "deleted"
	relatedItemIDField = "id"
	collectionType     = "story-package"
	contentURIPrefix   = "http://next-video-content-collection-mapper.svc.ft.com/content-collection/story-package/"
)

type relatedContentMapper struct {
	sc           serviceConfig
	strContent   string
	tid          string
	lastModified string
	unmarshalled map[string]interface{}
}

func (m *relatedContentMapper) mapRelatedContent() ([]byte, string, error) {
	videoUUID, err := getRequiredStringField(videoUUIDField, m.unmarshalled)
	if err != nil {
		return nil, "", err
	}

	var relatedItems []Item
	if !m.isDeleteEvent() {
		relatedItemsArray, err := getObjectsArrayField(relatedField, m.unmarshalled, videoUUID, m)
		if err != nil {
			return nil, videoUUID, err
		}

		relatedItems = m.retrieveRelatedItems(relatedItemsArray, videoUUID)
	}

	mc := m.buildMappedContent(videoUUID, relatedItems)

	marshalledPubEvent, err := json.Marshal(mc)
	if err != nil {
		logger.videoEvent(m.tid, videoUUID, "Error marshalling processed related items")
		return nil, videoUUID, err
	}

	return marshalledPubEvent, videoUUID, nil
}

func (m *relatedContentMapper) retrieveRelatedItems(relatedItemsArray []map[string]interface{}, videoUUID string) []Item {
	var result = make([]Item, 0)
	for _, relatedItem := range relatedItemsArray {
		itemID, err := getRequiredStringField(relatedItemIDField, relatedItem)
		if err != nil {
			logger.videoErrorEvent(m.tid, videoUUID, err, "Cannot extract related item id from related field")
			continue
		}
		result = append(result, Item{UUID: itemID})
	}
	return result
}

func (m *relatedContentMapper) buildMappedContent(videoUUID string, items []Item) MappedContent {
	collectionContainerUUID := NewNameUUIDFromBytes([]byte(videoUUID)).String()
	cc := ContentCollection{
		UUID:             collectionContainerUUID,
		Items:            items,
		PublishReference: m.tid,
		LastModified:     m.lastModified,
		CollectionType:   collectionType,
	}

	return MappedContent{
		Payload:      cc,
		ContentURI:   contentURIPrefix + collectionContainerUUID,
		LastModified: m.lastModified,
		UUID:         collectionContainerUUID,
	}
}

func getRequiredStringField(key string, obj map[string]interface{}) (string, error) {
	valueI, ok := obj[key]
	if !ok || valueI == nil {
		return "", nullFieldError(key)
	}

	val, ok := valueI.(string)
	if !ok {
		return "", wrongFieldTypeError("string", key, valueI)
	}
	return val, nil
}

func getObjectsArrayField(key string, obj map[string]interface{}, videoUUID string, vm *relatedContentMapper) ([]map[string]interface{}, error) {
	var objArrayI interface{}
	var result = make([]map[string]interface{}, 0)
	objArrayI, ok := obj[key]
	if !ok {
		logger.videoMapEvent(vm.tid, videoUUID, nullFieldError(key).Error())
		return result, nil
	}

	var objArray []interface{}
	objArray, ok = objArrayI.([]interface{})
	if !ok {
		return nil, wrongFieldTypeError("object array", key, objArrayI)
	}

	for _, objI := range objArray {
		obj, ok = objI.(map[string]interface{})
		if !ok {
			return nil, wrongFieldTypeError("object array", key, objArrayI)
		}
		result = append(result, obj)
	}
	return result, nil
}

func nullFieldError(fieldKey string) error {
	return fmt.Errorf("[%s] field of native Next video JSON is missing or is null", fieldKey)
}

func wrongFieldTypeError(expectedType, fieldKey string, value interface{}) error {
	return fmt.Errorf("[%s] field of native Next video JSON is not of type %s: [%v]", fieldKey, expectedType, value)
}

func (m *relatedContentMapper) isDeleteEvent() bool {
	if _, present := m.unmarshalled[deletedField]; present {
		return true
	}
	return false
}
