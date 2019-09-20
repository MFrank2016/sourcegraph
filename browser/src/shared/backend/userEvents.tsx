import { gql } from '../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../shared/src/graphql/schema'
import { PlatformContext } from '../../../../shared/src/platform/context'
import { DEFAULT_SOURCEGRAPH_URL } from '../util/context'

/**
 * Log a user action on the associated self-hosted Sourcegraph instance (allows site admins on a private
 * Sourcegraph instance to see a count of unique users on a daily, weekly, and monthly basis).
 *
 * This is never sent to Sourcegraph.com (i.e., when using the integration with open source code).
 *
 * @deprecated Use logEvent
 */
export const logUserEvent = (
    event: GQL.UserEvent,
    uid: string,
    url: string,
    requestGraphQL: PlatformContext['requestGraphQL']
): void => {
    // Only send the request if this is a private, self-hosted Sourcegraph instance.
    if (url === DEFAULT_SOURCEGRAPH_URL) {
        return
    }
    requestGraphQL<GQL.IMutation>({
        request: gql`
            mutation logUserEvent($event: UserEvent!, $userCookieID: String!) {
                logUserEvent(event: $event, userCookieID: $userCookieID) {
                    alwaysNil
                }
            }
        `,
        variables: { event, userCookieID: uid },
        mightContainPrivateInfo: false,
    }).subscribe({
        error: error => {
            // Swallow errors. If a Sourcegraph instance isn't upgraded, this request may fail
            // (e.g., if CODEINTELINTEGRATION user events aren't yet supported).
            // However, end users shouldn't experience this failure, as their admin is
            // responsible for updating the instance, and has been (or will be) notified
            // that an upgrade is available via site-admin messaging.
        },
    })
}

/**
 * Log a raw user action on the associated self-hosted Sourcegraph instance (allows site admins on a private
 * Sourcegraph instance to see a count of unique users on a daily, weekly, and monthly basis).
 *
 * This is never sent to Sourcegraph.com (i.e., when using the integration with open source code).
 */
export const logEvent = (
    event: { name: string; userCookieID: string; url: string },
    requestGraphQL: PlatformContext['requestGraphQL']
): void => {
    // Only send the request if this is a private, self-hosted Sourcegraph instance.
    if (event.url === DEFAULT_SOURCEGRAPH_URL) {
        return
    }

    requestGraphQL<GQL.IMutation>({
        request: gql`
            mutation logEvent($name: String!, $userCookieID: String!, $url: String!, $source: EventSource!) {
                logEvent(event: $name, userCookieID: $userCookieID, url: $url, source: $source) {
                    alwaysNil
                }
            }
        `,
        variables: {
            ...event,
            source: GQL.EventSource.CODEHOSTINTEGRATION,
        },
        mightContainPrivateInfo: false,
    }).subscribe({
        error: error => {
            // Swallow errors. If a Sourcegraph instance isn't upgraded, this request may fail
            // (i.e. the new GraphQL API `logEvent` hasn't been added).
            // However, end users shouldn't experience this failure, as their admin is
            // responsible for updating the instance, and has been (or will be) notified
            // that an upgrade is available via site-admin messaging.
        },
    })
}
