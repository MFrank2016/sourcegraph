import { Observable, BehaviorSubject, from } from 'rxjs'
import * as sourcegraph from 'sourcegraph'
import { switchMap, catchError, map, distinctUntilChanged } from 'rxjs/operators'
import { combineLatestOrDefault } from '../../../util/rxjs/combineLatestOrDefault'
import { isEqual, flatten, compact } from 'lodash'

/**
 * A service that manages and queries registered diagnostic providers
 * ({@link sourcegraph.DiagnosticProvider}).
 */
export interface DiagnosticService {
    /**
     * Observe the diagnostics provided by registered providers or by a specific provider.
     *
     * @param type Only observe diagnostics from the provider registered with this type. If
     * undefined, diagnostics from all providers are observed.
     */
    observeDiagnostics(
        scope: Parameters<sourcegraph.DiagnosticProvider['provideDiagnostics']>[0],
        type?: Parameters<typeof sourcegraph.workspace.registerDiagnosticProvider>[0]
    ): Observable<sourcegraph.Diagnostic[]>

    /**
     * Register a diagnostic provider.
     *
     * @returns An unsubscribable to unregister the provider.
     */
    registerDiagnosticProvider: typeof sourcegraph.workspace.registerDiagnosticProvider
}

/**
 * Creates a new {@link DiagnosticService}.
 */
export function createDiagnosticService(logErrors = true): DiagnosticService {
    interface Registration {
        type: Parameters<typeof sourcegraph.workspace.registerDiagnosticProvider>[0]
        provider: sourcegraph.DiagnosticProvider
    }
    const registrations = new BehaviorSubject<Registration[]>([])
    return {
        observeDiagnostics: (scope, type) => {
            return registrations.pipe(
                switchMap(registrations =>
                    combineLatestOrDefault(
                        (type === undefined ? registrations : registrations.filter(r => r.type === type)).map(
                            ({ provider }) =>
                                from(provider.provideDiagnostics(scope)).pipe(
                                    catchError(err => {
                                        if (logErrors) {
                                            console.error(err)
                                        }
                                        return [null]
                                    })
                                )
                        )
                    ).pipe(
                        map(itemsArrays => flatten(compact(itemsArrays))),
                        distinctUntilChanged((a, b) => isEqual(a, b))
                    )
                )
            )
        },
        registerDiagnosticProvider: (type, provider) => {
            if (registrations.value.some(r => r.type === type)) {
                throw new Error(`a DiagnosticProvider of type ${JSON.stringify(type)} is already registered`)
            }
            const reg: Registration = { type, provider }
            registrations.next([...registrations.value, reg])
            const unregister = () => registrations.next(registrations.value.filter(r => r !== reg))
            return { unsubscribe: unregister }
        },
    }
}
