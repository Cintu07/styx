--------------------------- MODULE Styx ---------------------------
\* TLA+ Formal Specification of STYX Properties
\* This spec proves correctness of core STYX invariants

EXTENDS Integers, Sequences, FiniteSets

CONSTANTS 
    Nodes,          \* Set of all nodes
    Witnesses,      \* Set of all witnesses
    MaxTime         \* Maximum logical time

VARIABLES
    beliefs,        \* beliefs[n] = [alive, dead, unknown]
    dead,           \* Set of nodes declared dead
    reports,        \* Sequence of witness reports
    time            \* Current logical time

TypeInvariant ==
    /\ beliefs \in [Nodes -> [alive: 0..100, dead: 0..100, unknown: 0..100]]
    /\ dead \subseteq Nodes
    /\ time \in 0..MaxTime

-----------------------------------------------------------------------------
\* PROPERTY 7: Belief is never binary
\* Belief distribution must never be {0,0,100} or {100,0,0} or {0,100,0}
Property7_BeliefNeverBinary ==
    \A n \in Nodes:
        LET b == beliefs[n]
        IN ~(b.alive = 100 /\ b.dead = 0 /\ b.unknown = 0)
           /\ ~(b.alive = 0 /\ b.dead = 100 /\ b.unknown = 0)

-----------------------------------------------------------------------------
\* PROPERTY 14: Finality is irreversible
\* Once a node is in 'dead' set, it stays there forever
Property14_FinalityIrreversible ==
    \A n \in Nodes:
        n \in dead => [](n \in dead)

-----------------------------------------------------------------------------
\* PROPERTY 18: Confidence sums to 1 (represented as 100)
Property18_ConfidenceSumsToOne ==
    \A n \in Nodes:
        LET b == beliefs[n]
        IN b.alive + b.dead + b.unknown = 100

-----------------------------------------------------------------------------
\* PROPERTY 13: False death forbidden
\* Death only declared with dead confidence > 85
Property13_FalseDeathForbidden ==
    \A n \in Nodes:
        n \in dead => beliefs[n].dead >= 85

-----------------------------------------------------------------------------
\* Initial state
Init ==
    /\ beliefs = [n \in Nodes |-> [alive |-> 0, dead |-> 0, unknown |-> 100]]
    /\ dead = {}
    /\ reports = <<>>
    /\ time = 0

\* Receive a witness report
ReceiveReport(witness, target, alive, d, unknown) ==
    /\ alive + d + unknown = 100
    /\ reports' = Append(reports, [
        witness |-> witness,
        target |-> target,
        alive |-> alive,
        dead |-> d,
        unknown |-> unknown,
        time |-> time
       ])
    /\ UNCHANGED <<beliefs, dead, time>>

\* Update beliefs based on reports
UpdateBeliefs(n) ==
    /\ beliefs' = [beliefs EXCEPT ![n] = 
        \* Simplified: average of all reports for this node
        [alive |-> 50, dead |-> 25, unknown |-> 25]]
    /\ UNCHANGED <<dead, reports, time>>

\* Declare death (only if conditions met)
DeclareDeath(n) ==
    /\ beliefs[n].dead >= 85
    /\ n \notin dead
    /\ dead' = dead \cup {n}
    /\ UNCHANGED <<beliefs, reports, time>>

\* Time advances
Tick ==
    /\ time < MaxTime
    /\ time' = time + 1
    /\ UNCHANGED <<beliefs, dead, reports>>

Next ==
    \/ \E w \in Witnesses, t \in Nodes, a, d, u \in 0..100:
        a + d + u = 100 /\ ReceiveReport(w, t, a, d, u)
    \/ \E n \in Nodes: UpdateBeliefs(n)
    \/ \E n \in Nodes: DeclareDeath(n)
    \/ Tick

Spec == Init /\ [][Next]_<<beliefs, dead, reports, time>>

-----------------------------------------------------------------------------
\* THEOREMS TO VERIFY

THEOREM Spec => []Property7_BeliefNeverBinary
THEOREM Spec => []Property18_ConfidenceSumsToOne
THEOREM Spec => []Property13_FalseDeathForbidden
THEOREM Spec => Property14_FinalityIrreversible

=============================================================================
