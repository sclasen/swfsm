/*
TaskPollers

DecisionTaskPoller and ActivityTaskPoller facilitate proper usage of the PollForDecisionTask and PollForActivityTask endpoints in the SWF API. These endpoints are used by
DecisionTask and ActivityTask workers to claim tasks on which to work. The endpoints use long polling. SWF will keep the request open for up to
60 seconds before returning an 'empty' response. If a task is generated before that time, a non-empty task is delivered (and assigned to) a particular
polling client.

There is an unfortunate bug in SWF that occurs when a long-polling request gets terminated client side, rather than waiting for the SWF API to respond.
SWF does not recognize this condition so it can result in assigning a task to a disconnected worker, which will subsequently cause the task to timeout.
This is not terrible if the task has a short timeout but can cause big delays if the task does have a long timeout.

Both types of pollers allow you to manage polling yourself by calling Poll() directly. However it is recommended that you use the
PollUntilShutdownBy(...) function, which works in concert with a PollerShutdownManager to await all in-flight polls to complete. This facilitates clean shutdown of end
user processes.

PollerShutdownManager

When PollerShutdownManager.ShutdownPollers() is called, it will signal any registered pollers to shut down
once any in-flight polls have completed, and block until this happens. The shutdown process can take up to 60 seconds
due to the length of SWF long polls before an empty response is returned.
*/
package poller


