# STYX Documentation

## Introduction

STYX is a distributed membership system designed to answer one fundamental question that every distributed system must answer: which nodes are alive and which nodes are dead. This question sounds simple but it is actually one of the hardest problems in distributed computing. Most existing systems get it wrong because they rely on mechanisms that fundamentally cannot provide correct answers in all situations.

The core insight behind STYX is that uncertainty is not a bug to be eliminated but a fundamental property of distributed systems that must be explicitly modeled and communicated. When a node stops responding to health checks, the correct answer is not "the node is dead" but rather "we do not have enough information to determine the state of the node with certainty." STYX makes this uncertainty explicit by representing node states as probability distributions rather than boolean values.

## The Problem with Existing Approaches

Traditional distributed systems use several mechanisms to detect node failures. The most common approach is timeout based detection where if a node does not respond within a configured time window it is declared dead. This approach is fundamentally flawed because network latency, server load, and garbage collection pauses can all cause temporary unresponsiveness that looks identical to actual failure. When the system incorrectly declares a healthy node as dead, it triggers recovery procedures that can cause cascading failures throughout the system.

Heartbeat based systems have a similar problem. A node sends periodic messages to indicate it is alive and if those messages stop arriving the node is assumed dead. But heartbeats can be delayed by network congestion. A node can also continue sending heartbeats while being functionally broken in ways that prevent it from doing useful work. The heartbeat mechanism cannot detect these failure modes.

Gossip protocols spread health information through the cluster by having nodes exchange information with random peers. While this provides better fault tolerance than centralized monitoring, gossip still ultimately produces boolean alive or dead decisions that do not capture the inherent uncertainty of the situation.

STYX takes a fundamentally different approach. Instead of producing boolean answers, STYX represents beliefs about node states as probability distributions over three possible states: alive, dead, and unknown. For any given node, STYX might report something like "61% confidence the node is alive, 19% confidence the node is dead, and 20% confidence we cannot determine the state." This representation allows consumers of the information to make their own decisions based on their risk tolerance rather than having that decision made for them by the membership system.

## Core Principles

STYX is built on several core principles that guide all design decisions.

The first principle is that uncertainty is first class. The system does not hide uncertainty or paper over it with arbitrary defaults. When STYX does not know the answer, it says so explicitly. This allows other components of the system to make informed decisions about how to proceed.

The second principle is that false death is worse than delayed death. Incorrectly declaring a healthy node as dead can cause significant damage to the system. Data might be rerouted unnecessarily, recovery procedures might be triggered, and load might be redistributed in ways that cause further problems. It is better to take more time to reach a confident conclusion than to make a fast decision that turns out to be wrong.

The third principle is that silence is better than lies. If STYX cannot provide an honest answer to a query, it will refuse to answer rather than provide information that might be misleading. This refusal is itself valuable information that tells the consumer that the system state is currently too uncertain for reliable decisions.

The fourth principle is that disagreement is preserved. When different witnesses have different views of a node's state, STYX does not simply pick a winner or average the results. It tracks the disagreement explicitly because disagreement often indicates network partitions or other systemic issues that are important to surface.

The fifth principle is that death is irreversible. Once STYX has declared a node dead with high confidence based on evidence from multiple witnesses, that declaration cannot be undone. This prevents the zombie node problem where a node that was correctly declared dead somehow comes back and causes confusion. If a node restarts after being declared dead, it must rejoin the cluster with a new identity.

## Architecture Overview

STYX is composed of several packages that each handle a specific responsibility. The packages are designed to be loosely coupled so that they can be tested independently and so that alternative implementations can be substituted if needed.

The types package defines the core data types used throughout the system. The most important type is Belief which represents a probability distribution over the alive, dead, and unknown states. A Belief always sums to 1.0 ensuring that the probabilities are normalized. The package also defines NodeID which uniquely identifies a node including a generation counter that is incremented when a node restarts, ensuring that a restarted node is treated as a new identity.

The time package provides logical timestamps based on Lamport clocks. Physical wall clock time is unreliable in distributed systems because clocks can drift and can even go backwards. Logical time provides a consistent ordering of events that does not depend on synchronized clocks.

The evidence package defines different types of evidence that can be used to infer node state. Evidence includes direct responses from the node, timeouts when the node does not respond, and observations of causal relationships between events. Each piece of evidence has a weight that indicates how strongly it suggests a particular state. Evidence also decays over time so that stale observations have less influence than recent ones.

The observer package implements single observer probing. An observer sends probes to target nodes and records the responses or timeouts as evidence. The observer also tracks local jitter which is the variance in its own scheduling delays. This is important because if the observer itself is under heavy load, timeouts might be caused by the observer's delays rather than the target node's failure. By tracking jitter, the observer can discount timeout evidence when local delays are high.

The witness package implements multi witness aggregation. Multiple observers can report their beliefs about a node and the witness package aggregates these reports into a combined belief. The aggregation takes into account witness trust which decays over time for witnesses that provide incorrect information. The package also detects correlation between witnesses because if multiple witnesses are running on the same physical machine their reports are not truly independent.

The finality package implements irreversible death declaration. Death can only be declared when there is overwhelming evidence from multiple independent witnesses. Once death is declared it cannot be undone. The package enforces the requirement that timeout evidence alone is not sufficient to declare death because a node might be unreachable due to network issues rather than actual failure.

The partition package detects network partitions by analyzing disagreement between witnesses. When some witnesses report a node as alive and others report it as dead, this often indicates a network partition where different parts of the cluster have different views. During a detected partition, STYX refuses to answer queries because any answer would be potentially misleading.

The oracle package ties everything together and provides the main API. The Oracle accepts queries for node state and witness reports about node observations. It integrates the observer, witness, finality, and partition packages to produce query results. The Oracle can also refuse to answer queries when conditions make honest answers impossible.

The api package provides an HTTP server that exposes the Oracle functionality over the network. The server provides endpoints for health checks, node queries, witness reports, and Prometheus compatible metrics.

## How Evidence Works

Evidence is the foundation of all beliefs in STYX. When an observer probes a target node and receives a response, it creates evidence that the node is alive. When a probe times out, it creates evidence that the node might be dead. But timeout evidence is treated more carefully than response evidence because timeouts have multiple possible causes.

Each piece of evidence has a weight between 0 and 1 that indicates how strongly it suggests a particular state. Direct response evidence has high weight because if a node responds to a probe it is definitely alive at that moment. Timeout evidence has lower weight because timeouts can be caused by network issues, observer delays, or temporary overload.

Evidence also decays over time. An observation from 10 seconds ago is more relevant than an observation from 10 minutes ago. The decay is exponential with a configurable half life. After one half life has elapsed, the evidence has half its original weight. After two half lives, it has one quarter of its original weight, and so on.

The evidence set for a node combines all available evidence using weighted aggregation. Evidence suggesting alive increases the alive component of the belief. Evidence suggesting dead increases the dead component. When there is little evidence or the evidence is conflicting, the unknown component increases.

## Jitter Awareness

A key innovation in STYX is jitter aware timeout handling. Traditional systems treat a timeout as strong evidence of failure. But timeouts can be caused by the observer's own delays rather than the target's failure.

STYX tracks scheduling jitter on the observer by measuring how long it takes for scheduled tasks to actually execute. If the observer is supposed to send a probe every 100 milliseconds but the probe is actually sent after 150 milliseconds due to garbage collection or CPU contention, then a 200 millisecond timeout has only 50 milliseconds of actual network wait time.

When local jitter is high, timeout evidence is discounted. The discount factor is based on the ratio of jitter to timeout. If jitter is 50% of the timeout, the timeout evidence is discounted by 50%. This prevents overloaded observers from incorrectly declaring nodes as dead.

## Multi Witness Aggregation

Single observer failure detection is unreliable because the observer might have a biased view of the network. STYX supports multiple witnesses that each independently observe target nodes and report their beliefs.

Witness reports are aggregated using trust weighted averaging. Each witness has a trust score that starts at a default value and is adjusted based on the accuracy of their reports. Witnesses that provide reports that are later confirmed as correct increase their trust. Witnesses that provide reports that turn out to be wrong decrease their trust.

Trust decay serves multiple purposes. It reduces the influence of compromised or buggy witnesses. It also handles the case where infrastructure changes cause a previously reliable witness to become unreliable.

The aggregation algorithm also detects correlation between witnesses. If multiple witnesses provide identical or nearly identical reports, they might be collocated on the same physical machine or connected to the same network segment. Correlated witnesses should not be counted as independent observations. When correlation is detected, the effective weight of each correlated witness is reduced.

## Partition Detection

Network partitions are one of the hardest problems in distributed systems. During a partition, different parts of the cluster cannot communicate with each other and might develop divergent views of the system state.

STYX detects partitions by analyzing disagreement between witnesses. If some witnesses report a node as alive with high confidence while others report it as dead with high confidence, this is a strong signal that the witnesses are on different sides of a partition.

When a partition is detected, STYX refuses to answer queries about affected nodes. This might seem unhelpful but it is actually the correct behavior. During a partition, any boolean answer would be wrong for some part of the cluster. By refusing to answer, STYX signals to the consumer that the situation requires human intervention or a more specialized recovery procedure.

## Finality and Irreversible Death

Death declaration in STYX is a serious action with permanent consequences. Once a node is declared dead, it cannot be undone.

Death can only be declared when multiple conditions are met. The aggregated belief must show very high confidence that the node is dead, typically 85% or higher. Multiple independent witnesses must agree on the death. The evidence must include non timeout evidence because timeouts alone are not sufficient proof of death. The disagreement between witnesses must be low.

These conditions are intentionally strict because false death declarations are very costly. If a node is incorrectly declared dead, all references to that node become invalid. Data owned by that node might be unnecessarily replicated or recovered. The node itself might continue operating unaware that the rest of the cluster considers it dead, potentially causing split brain scenarios.

When death is declared, the node's identity is permanently marked as dead. If the physical machine restarts, it must rejoin the cluster with a new identity. This is accomplished by including a generation counter in the NodeID. Each restart increments the generation, making the new instance distinguishable from the old one.

## The Oracle API

The Oracle is the main interface for using STYX. It provides a Query method that returns the current belief about a node's state.

The query result includes the belief distribution which gives the probabilities for alive, dead, and unknown states. It also includes metadata like the number of witnesses that contributed to the belief, the disagreement level between witnesses, and the current partition state.

The query result might indicate that the Oracle refused to answer. This happens during detected partitions or when the confidence requirements cannot be met. The refusal reason is included so the consumer understands why the answer was not provided.

Confidence requirements can be specified when making a query. For example, a consumer might require at least 70% confidence in the alive state and at most 20% unknown. If these requirements cannot be met, the Oracle refuses rather than providing an uncertain answer.

## HTTP API Endpoints

The HTTP API makes STYX accessible to any programming language that can make HTTP requests.

The health endpoint at /health returns a simple status indicating whether the server is running. This is intended for load balancer health checks and monitoring systems.

The query endpoint at /query accepts a target node ID and returns the full belief distribution and metadata. The response is JSON formatted and includes all the fields from the Oracle query result.

The report endpoint at /report accepts witness reports. A witness sends its ID, the target node ID, and its belief about the target. The beliefs must sum to 1.0 or the request is rejected.

The metrics endpoint at /metrics returns Prometheus formatted metrics including query counts, report counts, refusal counts, and latency information.

## Command Line Interface

STYX includes a command line tool for interacting with a running server.

The query command queries a node's status and displays the belief distribution in a human readable format. It also provides a simple status interpretation like "likely alive" or "uncertain".

The report command submits a witness report from the command line. This is useful for testing and debugging.

The health command checks whether the server is responding.

## Docker Deployment

STYX can be deployed using Docker. The Dockerfile uses a multi stage build to produce a small final image. The build stage compiles the Go code. The final stage is based on Alpine Linux and contains only the compiled binary and necessary certificates.

If Docker is not opening or responding when you try to run it, there are several possible causes. First, make sure Docker Desktop is installed and running on your machine. On Windows you can check the system tray for the Docker icon. If Docker Desktop is not running, start it and wait for it to fully initialize which can take a minute or two.

Second, make sure the Docker daemon is accessible. You can test this by running "docker version" in your terminal. If this command fails with a connection error, Docker is not running or not accessible to your user account.

Third, on Windows specifically, Docker Desktop requires WSL2 or Hyper V depending on your configuration. If neither is properly configured, Docker will not start. Check the Docker Desktop settings and logs for any error messages.

Fourth, if you are trying to access a container's port, make sure the port mapping is correct. The command "docker run -p 8080:8080 styx" maps port 8080 on your machine to port 8080 in the container. You should then be able to access the service at http://localhost:8080.

## Kubernetes Deployment

STYX includes Kubernetes manifests for deploying to a Kubernetes cluster. The deployment creates multiple replicas for high availability. The service exposes the deployment within the cluster. A headless service is also provided for use cases that need to discover individual pod addresses.

The Custom Resource Definition allows you to create StyxCluster resources that the operator will manage. This simplifies deployment and configuration.

## Byzantine Fault Tolerance

The byzantine package adds cryptographic protections against malicious witnesses. Each witness has a keypair and signs their reports. The aggregator verifies signatures before accepting reports.

Byzantine fault tolerance allows the system to operate correctly even when some witnesses are actively trying to provide false information. The system can tolerate up to one third of witnesses being malicious. This is achieved using a trimmed mean aggregation that discards the highest and lowest values before averaging.

## Failure Prediction

The prediction package uses historical data to predict failures before they happen. It tracks response times and failure events for each node.

The predictor looks for several patterns. If a node has not been seen recently, it predicts high failure probability. If response times are degrading compared to historical baselines, it predicts moderate failure probability. If there have been frequent recent failures, it predicts high failure probability.

This is not machine learning in the sense of neural networks or trained models. It is statistical anomaly detection based on simple heuristics. But these heuristics are effective for catching many common failure patterns.

## Economic Incentives

The economics package implements a staking and slashing mechanism for witnesses. Witnesses must stake tokens to participate. Correct reports earn rewards. Incorrect reports result in slashing where some staked tokens are forfeited.

This mechanism is designed for deployments where witnesses are operated by different parties who might have incentives to lie. By requiring a stake, witnesses have something to lose if they provide bad information. The economic incentives align witness behavior with system goals.

## Formal Verification

The formal directory contains a TLA+ specification of STYX. TLA+ is a formal specification language that allows mathematical verification of system properties.

The specification models the core state machine of STYX and proves several properties. Property 7 proves that beliefs are never binary. Property 13 proves that false death is forbidden. Property 14 proves that finality is irreversible. Property 18 proves that confidence values sum to one.

These proofs provide strong guarantees about system behavior that go beyond testing. A test can show that a property holds for specific inputs. A proof shows that the property holds for all possible inputs.

## Testing Approach

STYX is heavily tested with both unit tests and stress tests. The chaos package contains stress tests that push the system to its limits.

The standard chaos tests cover scenarios like byzantine witnesses where 30% of witnesses provide false information, flappy nodes that rapidly change state, timeout storms with many simultaneous timeouts, and partition chaos with repeated partition and heal cycles.

The god level tests push even harder. They test with 10000 nodes to verify scalability. They test with 50% byzantine witnesses to verify robustness. They test memory pressure to verify the system does not consume excessive memory. They test concurrent access to verify there are no race conditions.

## Conclusion

STYX represents a different approach to distributed membership that embraces uncertainty rather than hiding it. By representing beliefs as probability distributions, by tracking witness trust, by detecting partitions, and by treating death as irreversible, STYX provides a more honest and ultimately more useful foundation for distributed systems.

The system is production ready with comprehensive testing, Docker support, Kubernetes manifests, and multiple API options. It can be deployed as a standalone service or integrated as a library into existing Go applications.
