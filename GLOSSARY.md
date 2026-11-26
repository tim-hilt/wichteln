# Wichteln

A self-hosted Secret Santa app: a group of people secretly draws who gives a present to whom.

## People

**Organizer**:
The **Participant** who created a **Session**. The Organizer is always also a Participant.
_Avoid_: Admin, owner, host

**Participant**:
A person taking part in a **Session**, identified by name and email address.
_Avoid_: User, member, player

## Session

**Session**:
One Secret Santa round, consisting of its **Participants** and, eventually, their **Assignments**.
_Avoid_: Group, game, round, event

**Session link**:
The secret URL that gives the **Organizer** access to a **Session**'s overview.
_Avoid_: Admin link

**Personal link**:
The secret URL that gives one **Participant** access to their own **Wish**, **Exclusions** and, after the **Draw**, their **Assignment**.
_Avoid_: User link, invite link

## Wishes and drawing

**Wish**:
Free text in which a **Participant** describes what they would like to receive.
_Avoid_: Wishlist, gift idea

**Exclusion**:
A one-way rule set by a **Participant** that they must not give to a specific other **Participant**.
_Avoid_: Blocklist, constraint

**Submission**:
A **Participant** confirming their **Wish** and **Exclusions**. It can be edited until the **Draw**.

**Draw**:
The one-time event that randomly creates all **Assignments** of a **Session**, respecting all **Exclusions**.
_Avoid_: Matching, shuffle

**Assignment**:
The fact that one **Participant** gives a present to exactly one other **Participant**. Everyone gives once and receives once, and nobody knows who gives to them.
_Avoid_: Match, pair, partner
