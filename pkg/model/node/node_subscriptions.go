package node

// Subscription URLs are the stable, restart-safe way to record a node's configured
// data-flow targets: the peer base API URLs this node pushes events to. A node's own UUID
// changes on every restart (a fresh in-memory instance), but its URL is stable, so edges
// are keyed on URLs and resolved to the currently-live node by the graph consumer.

// HasSubscription reports whether the node already lists the given peer URL.
func HasSubscription(n Node, url string) bool {
	for _, existing := range n.GetSubscriptions() {
		if existing == url {
			return true
		}
	}
	return false
}

// AddSubscription adds a peer URL to the node's subscriptions if not already present,
// returning true when it was added. An empty URL or an already-present URL is a no-op.
func AddSubscription(n Node, url string) bool {
	if url == "" || HasSubscription(n, url) {
		return false
	}
	n.SetSubscriptions(append(n.GetSubscriptions(), url))
	return true
}

// RemoveSubscription drops a peer URL from the node's subscriptions and returns true when it
// was present.
func RemoveSubscription(n Node, url string) bool {
	current := n.GetSubscriptions()
	next := make([]string, 0, len(current))
	found := false
	for _, existing := range current {
		if existing == url {
			found = true
			continue
		}
		next = append(next, existing)
	}
	if found {
		n.SetSubscriptions(next)
	}
	return found
}
