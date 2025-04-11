## [Discovering Agent (Card)s](https://google.github.io/A2A/#/topics/agent_discovery?id=discovering-agent-cards)

- [Discovering Agent Cards](https://google.github.io/A2A/#/topics/agent_discovery?id=discovering-agent-cards)
  - [Open Discovery](https://google.github.io/A2A/#/topics/agent_discovery?id=open-discovery)
  - [Curated Discovery (Registry-Based)](https://google.github.io/A2A/#/topics/agent_discovery?id=curated-discovery-registry-based)
  - [Private Discovery (API-Based)](https://google.github.io/A2A/#/topics/agent_discovery?id=private-discovery-api-based)
  - [Securing Agent Cards](https://google.github.io/A2A/#/topics/agent_discovery?id=securing-agent-cards)

A2A's [AgentCard](https://google.github.io/A2A/#/documentation?id=agent-card) standardizes the _format_ of the data shared during discovery. However there are unlimited ways to discover these agent cards. We anticipate this being an open topic for discussion and look forward to ideas from the community.

Here is our current thinking.

## [Open Discovery](https://google.github.io/A2A/#/topics/agent_discovery?id=open-discovery)

We recommend enterprises host their agent cards at a well-known path. Specifically: https://`DOMAIN`/.well-known/agent.json. Clients will use DNS to resolve a known or found domain, send a simple `GET` request to the path, and receive the agent card.

This will enable web-crawlers and applications to easily discover agents for known or configured domains. This effectively reduces the discovery process to "find a domain".

## [Curated Discovery (Registry-Based)](https://google.github.io/A2A/#/topics/agent_discovery?id=curated-discovery-registry-based)

We anticipate enterprise applications making curated registries of agents available through a catalog interface. This opens up more enterprise scenarios such as company-specific or team-specific agent registries that are curated by an administrator.

_We **are** considering adding Registry support to the protocol - please drop us a [note](https://github.com/google/A2A/blob/main/README.md#contributing) with your opinion and where you see this being valuable as a standard_

## [Private Discovery (API-Based)](https://google.github.io/A2A/#/topics/agent_discovery?id=private-discovery-api-based)

There will undoubtably be private "agent stores" or proprietary agents where cards are exchanged behind custom APIs.

_We **are not** considering private discovery APIs as an A2A concern - please drop us a [note](https://github.com/google/A2A/blob/main/README.md#contributing) with your opinion and where you see this being valuable as a standard_

## [Securing Agent Cards](https://google.github.io/A2A/#/topics/agent_discovery?id=securing-agent-cards)

Agent cards may contain sensitive information. Implementors may decide to secure their agent cards behind controls that require authentication and authorization. For example, within an organization, even an open discovery at a well-known path could be guarded by mTLS and restricted to specific clients. Registries and Private Discovery APIs should require authentication and return different artifacts for different identities.

Note that implementors may include credential information (such as API Keys) in their Agent Cards. It is recommended that this information is NEVER available without Authentication.
