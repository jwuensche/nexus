package router

import (
	"testing"
	"time"

	"github.com/gammazero/nexus/wamp"
)

type testPeer struct {
	in chan wamp.Message
}

func newTestPeer() *testPeer {
	return &testPeer{
		in: make(chan wamp.Message, 1),
	}
}

func (p *testPeer) Send(msg wamp.Message)     { p.in <- msg }
func (p *testPeer) Recv() <-chan wamp.Message { return p.in }
func (p *testPeer) Close()                    { return }

func TestBasicSubscribe(t *testing.T) {
	// Test subscribing to a topic.
	broker := NewBroker(false, true).(*broker)
	subscriber := newTestPeer()
	sess := &Session{Peer: subscriber}
	testTopic := wamp.URI("nexus.test.topic")
	broker.Submit(sess, &wamp.Subscribe{Request: 123, Topic: testTopic})

	// Test that subscriber received SUBSCRIBED message
	rsp := <-sess.Recv()
	sub, ok := rsp.(*wamp.Subscribed)
	if !ok {
		t.Fatal("expected ", wamp.SUBSCRIBED, " got: ", rsp.MessageType())
	}
	subID := sub.Subscription
	if subID == 0 {
		t.Fatal("invalid suvscription ID")
	}

	// Check that broker created subscription.
	topic, ok := broker.subscriptions[subID]
	if !ok {
		t.Fatal("broker missing subscription")
	}
	if topic != testTopic {
		t.Fatal("subscription to wrong topic")
	}
	_, ok = broker.topicSubscribers[testTopic]
	if !ok {
		t.Fatal("broker missing subscribers for topic")
	}
	_, ok = broker.sessionSubIDSet[sess]
	if !ok {
		t.Fatal("broker missing subscriber ID for session")
	}

	// Test subscribing to same topic again.
	broker.Submit(sess, &wamp.Subscribe{Request: 123, Topic: testTopic})
	// Test that subscriber received SUBSCRIBED message
	rsp = <-sess.Recv()
	sub, ok = rsp.(*wamp.Subscribed)
	if !ok {
		t.Fatal("expected ", wamp.SUBSCRIBED, " got: ", rsp.MessageType())
	}
	// Should get same subscription ID.
	subID2 := sub.Subscription
	if subID2 != subID {
		t.Fatal("invalid suvscription ID")
	}
	if len(broker.subscriptions) != 1 {
		t.Fatal("broker has too many subscriptions")
	}
	if len(broker.topicSubscribers[testTopic]) != 1 {
		t.Fatal("too many subscribers to ", testTopic)
	}
	if len(broker.sessionSubIDSet[sess]) != 1 {
		t.Fatal("too many subscriptions for session")
	}

	// Test subscribing to different topic.
	testTopic2 := wamp.URI("nexus.test.topic2")
	broker.Submit(sess, &wamp.Subscribe{Request: 123, Topic: testTopic2})
	// Test that subscriber received SUBSCRIBED message
	rsp = <-sess.Recv()
	sub, ok = rsp.(*wamp.Subscribed)
	if !ok {
		t.Fatal("expected ", wamp.SUBSCRIBED, " got: ", rsp.MessageType())
	}
	subID2 = sub.Subscription
	if subID2 == subID {
		t.Fatal("wrong suvscription ID")
	}
	if len(broker.subscriptions) != 2 {
		t.Fatal("wrong number of subscriptions")
	}
	if len(broker.topicSubscribers[testTopic]) != 1 {
		t.Fatal("too many subscribers to ", testTopic)
	}
	if len(broker.topicSubscribers[testTopic2]) != 1 {
		t.Fatal("too many subscribers to ", testTopic2)
	}
	if len(broker.sessionSubIDSet[sess]) != 2 {
		t.Fatal("wrong number of subscriptions for session")
	}
}

func TestUnsubscribe(t *testing.T) {
	// Subscribe to topic
	broker := NewBroker(false, true).(*broker)
	subscriber := newTestPeer()
	sess := &Session{Peer: subscriber}
	testTopic := wamp.URI("nexus.test.topic")
	broker.Submit(sess, &wamp.Subscribe{Request: 123, Topic: testTopic})
	rsp := <-sess.Recv()
	subID := rsp.(*wamp.Subscribed).Subscription

	// Test unsubscribing from topic.
	broker.Submit(sess, &wamp.Unsubscribe{Request: 124, Subscription: subID})
	// Check that session received UNSUBSCRIBED message.
	rsp = <-sess.Recv()
	unsub, ok := rsp.(*wamp.Unsubscribed)
	if !ok {
		t.Fatal("expected ", wamp.UNSUBSCRIBED, " got: ", rsp.MessageType())
	}
	unsubID := unsub.Request
	if unsubID == 0 {
		t.Fatal("invalid unsibscribe ID")
	}
	// Check the broker removed subscription.
	if _, ok = broker.subscriptions[subID]; ok {
		t.Fatal("subscription still exists")
	}
	if _, ok = broker.topicSubscribers[testTopic]; ok {
		t.Fatal("topic subscriber still exists")
	}
	if _, ok = broker.sessionSubIDSet[sess]; ok {
		t.Fatal("session subscription ID set still exists")
	}
}

func TestRemove(t *testing.T) {
	// Subscribe to topic
	broker := NewBroker(false, true).(*broker)
	subscriber := newTestPeer()
	sess := &Session{Peer: subscriber}
	testTopic := wamp.URI("nexus.test.topic")
	broker.Submit(sess, &wamp.Subscribe{Request: 123, Topic: testTopic})
	rsp := <-sess.Recv()
	subID := rsp.(*wamp.Subscribed).Subscription

	testTopic2 := wamp.URI("nexus.test.topic2")
	broker.Submit(sess, &wamp.Subscribe{Request: 456, Topic: testTopic2})
	rsp = <-sess.Recv()
	subID2 := rsp.(*wamp.Subscribed).Subscription

	broker.RemoveSession(sess)
	broker.sync()

	// Check the broker removed subscription.
	_, ok := broker.subscriptions[subID]
	if ok {
		t.Fatal("subscription still exists")
	}
	if _, ok = broker.topicSubscribers[testTopic]; ok {
		t.Fatal("topic subscriber still exists")
	}
	if _, ok = broker.subscriptions[subID2]; ok {
		t.Fatal("subscription still exists")
	}
	if _, ok = broker.topicSubscribers[testTopic2]; ok {
		t.Fatal("topic subscriber still exists")
	}
	if _, ok = broker.sessionSubIDSet[sess]; ok {
		t.Fatal("session subscription ID set still exists")
	}
}

func TestBasicPubSub(t *testing.T) {
	broker := NewBroker(false, true).(*broker)
	subscriber := newTestPeer()
	sess := &Session{Peer: subscriber}
	testTopic := wamp.URI("nexus.test.topic")
	msg := &wamp.Subscribe{
		Request: 123,
		Topic:   testTopic,
	}
	broker.Submit(sess, msg)

	// Test that subscriber received SUBSCRIBED message
	rsp := <-sess.Recv()
	_, ok := rsp.(*wamp.Subscribed)
	if !ok {
		t.Fatal("expected ", wamp.SUBSCRIBED, " got: ", rsp.MessageType())
	}

	publisher := newTestPeer()
	pubSess := &Session{Peer: publisher}
	broker.Submit(pubSess, &wamp.Publish{Request: 124, Topic: testTopic,
		Arguments: []interface{}{"hello world"}})
	rsp = <-sess.Recv()
	evt, ok := rsp.(*wamp.Event)
	if !ok {
		t.Fatal("expected", wamp.EVENT, "got:", rsp.MessageType())
	}
	if len(evt.Arguments) == 0 {
		t.Fatal("missing event payload")
	}
	if evt.Arguments[0].(string) != "hello world" {
		t.Fatal("wrong argument value in payload:", evt.Arguments[0])
	}
}

// ----- WAMP v.2 Testing -----

func TestPrefxPatternBasedSubscription(t *testing.T) {
	// Test match=prefix
	broker := NewBroker(false, true).(*broker)
	subscriber := newTestPeer()
	sess := &Session{Peer: subscriber}
	testTopic := wamp.URI("nexus.test.topic")
	testTopicPfx := wamp.URI("nexus.test.")
	msg := &wamp.Subscribe{
		Request: 123,
		Topic:   testTopicPfx,
		Options: map[string]interface{}{"match": "prefix"},
	}
	broker.Submit(sess, msg)

	// Test that subscriber received SUBSCRIBED message
	rsp := <-sess.Recv()
	sub, ok := rsp.(*wamp.Subscribed)
	if !ok {
		t.Fatal("expected ", wamp.SUBSCRIBED, " got: ", rsp.MessageType())
	}
	subID := sub.Subscription
	if subID == 0 {
		t.Fatal("invalid suvscription ID")
	}

	// Check that broker created subscription.
	topic, ok := broker.pfxSubscriptions[subID]
	if !ok {
		t.Fatal("broker missing subscription")
	}
	if topic != testTopicPfx {
		t.Fatal("subscription to wrong topic")
	}
	_, ok = broker.pfxTopicSubscribers[testTopicPfx]
	if !ok {
		t.Fatal("broker missing subscribers for topic")
	}
	_, ok = broker.sessionSubIDSet[sess]
	if !ok {
		t.Fatal("broker missing subscriber ID for session")
	}

	publisher := newTestPeer()
	pubSess := &Session{Peer: publisher}
	broker.Submit(pubSess, &wamp.Publish{Request: 124, Topic: testTopic})
	rsp = <-sess.Recv()
	evt, ok := rsp.(*wamp.Event)
	if !ok {
		t.Fatal("expected", wamp.EVENT, "got:", rsp.MessageType())
	}
	_topic, ok := evt.Details["topic"]
	if !ok {
		t.Fatalf("event missing topic")
	}
	topic = _topic.(wamp.URI)
	if topic != testTopic {
		t.Fatal("wrong topic received")
	}
}

func TestWildcardPatternBasedSubscription(t *testing.T) {
	// Test match=prefix
	broker := NewBroker(false, true).(*broker)
	subscriber := newTestPeer()
	sess := &Session{Peer: subscriber}
	testTopic := wamp.URI("nexus.test.topic")
	testTopicWc := wamp.URI("nexus..topic")
	msg := &wamp.Subscribe{
		Request: 123,
		Topic:   testTopicWc,
		Options: map[string]interface{}{"match": "wildcard"},
	}
	broker.Submit(sess, msg)

	// Test that subscriber received SUBSCRIBED message
	rsp := <-sess.Recv()
	sub, ok := rsp.(*wamp.Subscribed)
	if !ok {
		t.Fatal("expected ", wamp.SUBSCRIBED, " got: ", rsp.MessageType())
	}
	subID := sub.Subscription
	if subID == 0 {
		t.Fatal("invalid suvscription ID")
	}

	// Check that broker created subscription.
	topic, ok := broker.wcSubscriptions[subID]
	if !ok {
		t.Fatal("broker missing subscription")
	}
	if topic != testTopicWc {
		t.Fatal("subscription to wrong topic")
	}
	_, ok = broker.wcTopicSubscribers[testTopicWc]
	if !ok {
		t.Fatal("broker missing subscribers for topic")
	}
	_, ok = broker.sessionSubIDSet[sess]
	if !ok {
		t.Fatal("broker missing subscriber ID for session")
	}

	publisher := newTestPeer()
	pubSess := &Session{Peer: publisher}
	broker.Submit(pubSess, &wamp.Publish{Request: 124, Topic: testTopic})
	rsp = <-sess.Recv()
	evt, ok := rsp.(*wamp.Event)
	if !ok {
		t.Fatal("expected", wamp.EVENT, "got:", rsp.MessageType)
	}
	_topic, ok := evt.Details["topic"]
	if !ok {
		t.Fatalf("event missing topic")
	}
	topic = _topic.(wamp.URI)
	if topic != testTopic {
		t.Fatal("wrong topic received")
	}
}

func TestSubscriberBlackwhiteListing(t *testing.T) {
	broker := NewBroker(false, true).(*broker)
	subscriber := newTestPeer()
	sess := &Session{
		Peer:     subscriber,
		ID:       wamp.GlobalID(),
		AuthID:   "jdoe",
		AuthRole: "admin",
	}
	testTopic := wamp.URI("nexus.test.topic")

	broker.Submit(sess, &wamp.Subscribe{Request: 123, Topic: testTopic})

	// Test that subscriber received SUBSCRIBED message
	rsp := <-sess.Recv()
	_, ok := rsp.(*wamp.Subscribed)
	if !ok {
		t.Fatal("expected ", wamp.SUBSCRIBED, " got: ", rsp.MessageType())
	}

	publisher := newTestPeer()
	featFlag := map[string]interface{}{"subscriber_blackwhite_listing": true}
	featMap := map[string]interface{}{"features": featFlag}
	pubFeat := map[string]map[string]interface{}{"publisher": featMap}
	pubSess := &Session{Peer: publisher,
		Details: map[string]interface{}{"roles": pubFeat},
	}
	// Test whilelist
	broker.Submit(pubSess, &wamp.Publish{
		Request: 124,
		Topic:   testTopic,
		Options: map[string]interface{}{"eligible": []string{string(sess.ID)}},
	})
	rsp, err := wamp.RecvTimeout(sess, time.Millisecond)
	if err != nil {
		t.Fatal("not allowed by whitelist")
	}
	// Test whitelist authrole
	broker.Submit(pubSess, &wamp.Publish{
		Request: 125,
		Topic:   testTopic,
		Options: map[string]interface{}{"eligible_authrole": []string{"admin"}},
	})
	rsp, err = wamp.RecvTimeout(sess, time.Millisecond)
	if err != nil {
		t.Fatal("not allowed by authrole whitelist")
	}
	// Test whitelist authid
	broker.Submit(pubSess, &wamp.Publish{
		Request: 126,
		Topic:   testTopic,
		Options: map[string]interface{}{"eligible_authid": []string{"jdoe"}},
	})
	rsp, err = wamp.RecvTimeout(sess, time.Millisecond)
	if err != nil {
		t.Fatal("not allowed by authid whitelist")
	}

	// Test blacklist.
	broker.Submit(pubSess, &wamp.Publish{
		Request: 127,
		Topic:   testTopic,
		Options: map[string]interface{}{"exclude": []string{string(sess.ID)}},
	})
	rsp, err = wamp.RecvTimeout(sess, time.Millisecond)
	if err == nil {
		t.Fatal("not excluded by blacklist")
	}
	// Test blacklist authrole
	broker.Submit(pubSess, &wamp.Publish{
		Request: 128,
		Topic:   testTopic,
		Options: map[string]interface{}{"exclude_authrole": []string{"admin"}},
	})
	rsp, err = wamp.RecvTimeout(sess, time.Millisecond)
	if err == nil {
		t.Fatal("not excluded by authrole blacklist")
	}
	// Test blacklist authid
	broker.Submit(pubSess, &wamp.Publish{
		Request: 129,
		Topic:   testTopic,
		Options: map[string]interface{}{"exclude_authid": []string{"jdoe"}},
	})
	rsp, err = wamp.RecvTimeout(sess, time.Millisecond)
	if err == nil {
		t.Fatal("not excluded by authid blacklist")
	}

	// Test that blacklist takes precedence over whitelist.
	broker.Submit(pubSess, &wamp.Publish{
		Request: 126,
		Topic:   testTopic,
		Options: map[string]interface{}{"eligible_authid": []string{"jdoe"},
			"exclude_authid": []string{"jdoe"}},
	})
	rsp, err = wamp.RecvTimeout(sess, time.Millisecond)
	if err == nil {
		t.Fatal("should have been excluded by blacklist")
	}
}

func TestPublisherExclusion(t *testing.T) {
	broker := NewBroker(false, true).(*broker)
	subscriber := newTestPeer()
	sess := &Session{Peer: subscriber}
	testTopic := wamp.URI("nexus.test.topic")

	broker.Submit(sess, &wamp.Subscribe{Request: 123, Topic: testTopic})

	// Test that subscriber received SUBSCRIBED message
	rsp, err := wamp.RecvTimeout(sess, time.Millisecond)
	if err != nil {
		t.Fatal("subscribe session did not get response to SUBSCRIBE")
	}
	_, ok := rsp.(*wamp.Subscribed)
	if !ok {
		t.Fatal("expected ", wamp.SUBSCRIBED, " got: ", rsp.MessageType())
	}

	publisher := newTestPeer()
	featFlag := map[string]interface{}{"publisher_exclusion": true}
	featMap := map[string]interface{}{"features": featFlag}
	pubFeat := map[string]map[string]interface{}{"publisher": featMap}
	pubSess := &Session{Peer: publisher,
		Details: map[string]interface{}{"roles": pubFeat},
	}
	// Subscribe the publish session also.
	broker.Submit(pubSess, &wamp.Subscribe{Request: 123, Topic: testTopic})
	// Test that pub session received SUBSCRIBED message
	rsp, err = wamp.RecvTimeout(pubSess, time.Millisecond)
	if err != nil {
		t.Fatal("publish session did not get response to SUBSCRIBE")
	}
	_, ok = rsp.(*wamp.Subscribed)
	if !ok {
		t.Fatal("expected ", wamp.SUBSCRIBED, " got: ", rsp.MessageType())
	}

	// Publish message with exclud_me = false.
	broker.Submit(pubSess, &wamp.Publish{
		Request: 124,
		Topic:   testTopic,
		Options: map[string]interface{}{"exclude_me": false},
	})
	rsp, err = wamp.RecvTimeout(sess, time.Millisecond)
	if err != nil {
		t.Fatal("subscriber did not receive event")
	}
	rsp, err = wamp.RecvTimeout(pubSess, time.Millisecond)
	if err != nil {
		t.Fatal("pub session should have received event")
	}

	// Publish message with exclud_me = true.
	broker.Submit(pubSess, &wamp.Publish{
		Request: 124,
		Topic:   testTopic,
		Options: map[string]interface{}{"exclude_me": true},
	})
	rsp, err = wamp.RecvTimeout(sess, time.Millisecond)
	if err != nil {
		t.Fatal("subscriber did not receive event")
	}
	rsp, err = wamp.RecvTimeout(pubSess, time.Millisecond)
	if err == nil {
		t.Fatal("pub session should NOT have received event")
	}
}

func TestPublisherIdentification(t *testing.T) {
	broker := NewBroker(false, true).(*broker)
	subscriber := newTestPeer()
	featFlag := map[string]interface{}{"publisher_identification": true}
	featMap := map[string]interface{}{"features": featFlag}
	subFeat := map[string]map[string]interface{}{"subscriber": featMap}
	sess := &Session{Peer: subscriber,
		Details: map[string]interface{}{"roles": subFeat},
	}
	testTopic := wamp.URI("nexus.test.topic")

	broker.Submit(sess, &wamp.Subscribe{Request: 123, Topic: testTopic})

	// Test that subscriber received SUBSCRIBED message
	rsp := <-sess.Recv()
	_, ok := rsp.(*wamp.Subscribed)
	if !ok {
		t.Fatal("expected ", wamp.SUBSCRIBED, " got: ", rsp.MessageType())
	}

	publisher := newTestPeer()
	pubSess := &Session{Peer: publisher, ID: wamp.GlobalID()}
	broker.Submit(pubSess, &wamp.Publish{
		Request: 124,
		Topic:   testTopic,
		Options: map[string]interface{}{"disclose_me": true},
	})
	rsp = <-sess.Recv()
	evt, ok := rsp.(*wamp.Event)
	if !ok {
		t.Fatal("expected", wamp.EVENT, "got:", rsp.MessageType())
	}
	pub, ok := evt.Details["publisher"]
	if !ok {
		t.Fatal("missing publisher ID")
	}
	if pub.(wamp.ID) != pubSess.ID {
		t.Fatal("incorrect publisher ID disclosed")
	}
}
