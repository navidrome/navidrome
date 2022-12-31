This folder abstracts metadata lookup into "agents". Each agent can be implemented to get as
much info as the external source provides, by using a granular set of interfaces 
(see [interfaces](interfaces.go)).

A new agent must comply with these simple implementation rules:
1) Implement the `AgentName()` method. It just returns the name of the agent for logging purposes.
2) Implement one or more of the `*Retriever()` interfaces. That's where the agent's logic resides.
3) Register itself  (in its `init()` function).

For an agent to be used it needs to be listed in the `Agents` config option (default is `"lastfm,spotify"`). The order dictates the priority of the agents

For a simple Agent example, look at the [local_agent](local_agent.go) agent source code.
