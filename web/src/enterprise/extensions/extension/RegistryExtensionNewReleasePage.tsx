import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import H from 'history'
import ErrorIcon from 'mdi-react/ErrorIcon'
import React, { useCallback, useState } from 'react'
import { map, catchError, tap, concatMap } from 'rxjs/operators'
import { ConfiguredRegistryExtension } from '../../../../../shared/src/extensions/extension'
import { ExtensionManifest } from '../../../../../shared/src/extensions/extensionManifest'
import { gql, dataOrThrowErrors } from '../../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../../shared/src/graphql/schema'
import extensionSchemaJSON from '../../../../../shared/src/schema/extension.schema.json'
import { asError, isErrorLike } from '../../../../../shared/src/util/errors'
import { withAuthenticatedUser } from '../../../auth/withAuthenticatedUser'
import { mutateGraphQL } from '../../../backend/graphql'
import { Form } from '../../../components/Form'
import { HeroPage } from '../../../components/HeroPage'
import { PageTitle } from '../../../components/PageTitle'
import { DynamicallyImportedMonacoSettingsEditor } from '../../../settings/DynamicallyImportedMonacoSettingsEditor'
import { useLocalStorage } from '../../../util/useLocalStorage'
import { useEventObservable } from '../../../../../shared/src/util/useObservable'
import { fromFetch } from '../../../../../shared/src/graphql/fromFetch'
import { of, Observable, concat, from } from 'rxjs'
import { ErrorAlert } from '../../../components/alerts'
import CheckCircleIcon from 'mdi-react/CheckCircleIcon'

const publishExtension = (
    args: Pick<GQL.IPublishExtensionOnExtensionRegistryMutationArguments, 'extensionID' | 'manifest' | 'bundle'>
): Promise<GQL.IExtensionRegistryCreateExtensionResult> =>
    mutateGraphQL(
        gql`
            mutation PublishRegistryExtension($extensionID: String!, $manifest: String!, $bundle: String!) {
                extensionRegistry {
                    publishExtension(extensionID: $extensionID, manifest: $manifest, bundle: $bundle) {
                        extension {
                            url
                        }
                    }
                }
            }
        `,
        args
    )
        .pipe(
            map(dataOrThrowErrors),
            map(data => data.extensionRegistry.publishExtension)
        )
        .toPromise()

interface Props {
    /** The extension that is the subject of the page. */
    extension: ConfiguredRegistryExtension<GQL.IRegistryExtension>

    authenticatedUser: GQL.IUser
    isLightTheme: boolean
    history: H.History
}

const DEFAULT_MANIFEST: Pick<ExtensionManifest, 'activationEvents'> = {
    activationEvents: ['*'],
}

const LOADING = 'loading' as const

const DEFAULT_SOURCE = `const sourcegraph = require('sourcegraph')

function activate(context) {
    sourcegraph.app.activeWindow.showNotification(
        'Hello World!',
        sourcegraph.NotificationType.Success
    )
}

module.exports = { activate }
`

/** A page for publishing a new release of an extension to the extension registry. */
export const RegistryExtensionNewReleasePage = withAuthenticatedUser<Props>(({ extension, isLightTheme, history }) => {
    // Omit the `url` field from the extension so that it gets set to the URL of the bundle we're uploading.
    const manifestWithoutUrl = extension.rawManifest ? JSON.parse(extension.rawManifest) : { ...DEFAULT_MANIFEST }
    delete manifestWithoutUrl.url
    const [manifest, setManifest] = useState(JSON.stringify(manifestWithoutUrl, null, 2))

    const [onChangeBundle, bundleOrError] = useEventObservable(
        useCallback(
            (bundleChanges: Observable<string>) =>
                concat(
                    isErrorLike(extension.manifest) || !extension.manifest?.url
                        ? of(DEFAULT_SOURCE)
                        : fromFetch(extension.manifest.url, undefined, resp => resp.text()).pipe(
                              catchError(err => [asError(err)])
                          ),
                    bundleChanges
                ),
            [extension.manifest]
        )
    )

    const [onSubmit, updateOrError] = useEventObservable(
        useCallback(
            (submits: Observable<React.FormEvent>) =>
                submits.pipe(
                    tap(event => event.preventDefault()),
                    concatMap(() => {
                        if (isErrorLike(bundleOrError)) {
                            throw new Error('Invalid bundle')
                        }
                        return concat(
                            [LOADING],
                            from(publishExtension({ extensionID: extension.id, manifest, bundle: bundleOrError })).pipe(
                                catchError(err => [asError(err)])
                            )
                        )
                    })
                ),
            [bundleOrError, extension.id, manifest]
        )
    )

    const [showEditor, setShowEditor] = useLocalStorage('RegistryExtensionNewReleasePage.showEditor', false)
    const onShowEditorClick = useCallback(() => setShowEditor(true), [setShowEditor])

    return !extension.registryExtension || !extension.registryExtension.viewerCanAdminister ? (
        <HeroPage
            icon={ErrorIcon}
            title="Unauthorized"
            subtitle="You are not authorized to adminster this extension."
        />
    ) : (
        <div className="registry-extension-new-release-page">
            <PageTitle title="Publish new release" />
            <h2>Publish new release</h2>
            <p>
                Use the{' '}
                <a href="https://github.com/sourcegraph/src-cli" target="_blank" rel="noopener noreferrer">
                    <code>src</code> CLI tool
                </a>{' '}
                to publish a new release:
            </p>
            <pre>
                <code>$ src extensions publish</code>
            </pre>
            {showEditor ? (
                <>
                    <hr className="my-4" />
                    <h2>Extension editor (experimental)</h2>
                    <p>
                        Edit or paste in an extension JSON manifest and JavaScript bundle. The JavaScript bundle source
                        must be self-contained; dependency bundling and TypeScript transpilation is not yet supported.
                    </p>
                    <Form onSubmit={onSubmit} className="mb-3">
                        <div className="row">
                            <div className="col-lg-6">
                                <div className="form-group">
                                    <label htmlFor="registry-extension-new-release-page__manifest">
                                        <h3>Manifest</h3>
                                    </label>
                                    <DynamicallyImportedMonacoSettingsEditor
                                        id="registry-extension-new-release-page__manifest"
                                        className="d-block"
                                        value={manifest}
                                        onChange={setManifest}
                                        jsonSchema={extensionSchemaJSON}
                                        readOnly={updateOrError === LOADING}
                                        isLightTheme={isLightTheme}
                                        history={history}
                                    />
                                </div>
                            </div>
                            <div className="col-lg-6">
                                <div className="form-group">
                                    <label htmlFor="registry-extension-new-release-page__bundle">
                                        <h3>Source</h3>
                                    </label>
                                    {bundleOrError === undefined ? (
                                        <div>
                                            <LoadingSpinner className="icon-inline" />
                                        </div>
                                    ) : isErrorLike(bundleOrError) ? (
                                        <ErrorAlert error={bundleOrError} />
                                    ) : (
                                        <DynamicallyImportedMonacoSettingsEditor
                                            id="registry-extension-new-release-page__bundle"
                                            language="javascript"
                                            className="d-block"
                                            // Only 1 component can block navigation, and the
                                            // other editor does, so we don't.
                                            blockNavigationIfDirty={false}
                                            value={bundleOrError}
                                            onChange={onChangeBundle}
                                            readOnly={updateOrError === LOADING}
                                            isLightTheme={isLightTheme}
                                            history={history}
                                        />
                                    )}
                                </div>
                            </div>
                        </div>
                        <div className="d-flex align-items-center">
                            <button
                                type="submit"
                                disabled={updateOrError === LOADING || isErrorLike(bundleOrError)}
                                className="btn btn-primary mr-2"
                            >
                                Publish
                            </button>{' '}
                            {updateOrError &&
                                !isErrorLike(updateOrError) &&
                                (updateOrError === LOADING ? (
                                    <LoadingSpinner className="icon-inline" />
                                ) : (
                                    <span className="text-success">
                                        <CheckCircleIcon className="icon-inline" /> Published release successfully.
                                    </span>
                                ))}
                        </div>
                        {isErrorLike(updateOrError) && <ErrorAlert error={updateOrError} />}
                    </Form>
                </>
            ) : (
                <button type="button" className="btn btn-secondary" onClick={onShowEditorClick}>
                    Experimental: Use in-browser extension editor
                </button>
            )}
        </div>
    )
})
