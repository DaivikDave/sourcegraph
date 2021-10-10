import classNames from 'classnames'
import ChevronLeftIcon from 'mdi-react/ChevronLeftIcon'
import React from 'react'
import { Link, useLocation } from 'react-router-dom'

import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { ThemeProps } from '@sourcegraph/shared/src/theme'

import { BrandLogo } from '../components/branding/BrandLogo'
import { FeatureFlagProps } from '../featureFlags/featureFlags'
import { SourcegraphContext } from '../jscontext'

import styles from './CloudSignUpPage.module.scss'
import { OrDivider } from './OrDivider'
import { ExternalsAuth } from './SignInSignUpCommon'
import { SignUpArguments, SignUpForm } from './SignUpForm'

interface Props extends ThemeProps, TelemetryProps, FeatureFlagProps {
    source: string | null
    showEmailForm: boolean
    /** Called to perform the signup on the server. */
    onSignUp: (args: SignUpArguments) => Promise<void>
    context: Pick<SourcegraphContext, 'authProviders' | 'experimentalFeatures'>
}

const SourceToTitleMap = {
    Context: 'Easily search the code you care about, for free',
    Saved: 'Create a library of useful searches.',
    Monitor: 'Monitor code for changes.',
    Extend: 'Augment code and workflows via extensions.',
    SearchCTA: 'Easily search the code you care about, for free',
    HomepageCTA: 'Easily search the code you care about, for free',
    Snippet: 'Easily search the code you care about, for free',
}

export type CloudSignUpSource = keyof typeof SourceToTitleMap

export const ShowEmailFormQueryParameter = 'showEmail'

/**
 * Sign up page specifically for Sourcegraph.com
 */
export const CloudSignUpPage: React.FunctionComponent<Props> = ({
    isLightTheme,
    source,
    showEmailForm,
    onSignUp,
    context,
    telemetryService,
    featureFlags,
}) => {
    const location = useLocation()
    const isSignupOptimised = featureFlags.get('signup-optimization')

    const queryWithUseEmailToggled = new URLSearchParams(location.search)
    if (showEmailForm) {
        queryWithUseEmailToggled.delete(ShowEmailFormQueryParameter)
    } else {
        queryWithUseEmailToggled.append(ShowEmailFormQueryParameter, 'true')
    }

    const assetsRoot = window.context?.assetsRoot || ''
    const sourceIsValid = source && Object.keys(SourceToTitleMap).includes(source)
    const title = sourceIsValid ? SourceToTitleMap[source as CloudSignUpSource] : SourceToTitleMap.Context // Use Context as default

    const logEvent = (type: string): void => {
        if (sourceIsValid) {
            telemetryService.log('SignupInitiated', { type }, { type })
        }
    }

    return (
        <div className={styles.page}>
            <header>
                <div className="position-relative">
                    <div className={styles.headerBackground1} />
                    <div className={styles.headerBackground2} />
                    <div className={styles.headerBackground3} />

                    <div className={styles.limitWidth}>
                        <BrandLogo isLightTheme={isLightTheme} variant="logo" className={styles.logo} />
                    </div>
                </div>

                <div className={styles.limitWidth}>
                    <h2 className={styles.pageHeading}>{title}</h2>
                </div>
            </header>

            <div className={classNames(styles.contents, styles.limitWidth)}>
                <div className={styles.contentsLeft}>
                    With a Sourcegraph account, you can also:
                    <ul className={styles.featureList}>
                        <li>
                            <div className="d-flex align-items-center">
                                <span className="badge badge-info text-uppercase mr-1">Beta</span> Search across all
                                your public and private repositories
                            </div>
                        </li>
                        <li>Monitor code for changes</li>
                        <li>Navigate through code with IDE like go to references and definition hovers</li>
                        <li>Integrate data, tooling, and code in a single location </li>
                    </ul>
                    <div className={styles.companiesHeader}>
                        Trusted by developers at the world's most innovative companies:
                    </div>
                    <img
                        src={`${assetsRoot}/img/customer-logos-${isLightTheme ? 'light' : 'dark'}.svg`}
                        alt="Cloudflare, Uber, SoFi, Dropbox, Plaid, Toast"
                    />
                </div>

                <div className={styles.signUpWrapper}>
                    <h2>Create a free account</h2>
                    {isSignupOptimised ? (
                        <>
                            <SignUpForm
                                featureFlags={featureFlags}
                                onSignUp={args => {
                                    logEvent('form')
                                    return onSignUp(args)
                                }}
                                context={{ authProviders: [], sourcegraphDotComMode: true }}
                                buttonLabel="Sign up"
                                experimental={true}
                                className="my-3"
                            />
                            <div className={classNames('d-flex justify-content-center', styles.helperText)}>
                                <span className="mr-1"> Have an account?</span>
                                <Link to={`/sign-in${location.search}`}> Log in</Link>
                            </div>

                            <OrDivider className="mt-4 mb-4 text-lowercase" />

                            <ExternalsAuth
                                withCenteredText={true}
                                context={context}
                                githubLabel="Sign up with GitHub"
                                gitlabLabel="Sign up with GitLab"
                                onClick={logEvent}
                            />
                        </>
                    ) : !showEmailForm ? (
                        <>
                            <ExternalsAuth
                                context={context}
                                githubLabel="Continue with GitHub"
                                gitlabLabel="Continue with GitLab"
                                onClick={logEvent}
                            />

                            <div className="mb-4">
                                Or,{' '}
                                <Link to={`${location.pathname}?${queryWithUseEmailToggled.toString()}`}>
                                    continue with email
                                </Link>
                            </div>
                        </>
                    ) : (
                        <>
                            <small className="d-block mt-3">
                                <Link
                                    className="d-flex align-items-center"
                                    to={`${location.pathname}?${queryWithUseEmailToggled.toString()}`}
                                >
                                    <ChevronLeftIcon className={classNames('icon-inline', styles.backIcon)} />
                                    Go back
                                </Link>
                            </small>

                            <SignUpForm
                                featureFlags={featureFlags}
                                onSignUp={args => {
                                    logEvent('form')
                                    return onSignUp(args)
                                }}
                                context={{ authProviders: [], sourcegraphDotComMode: true }}
                                buttonLabel="Sign up"
                                experimental={true}
                                className="my-3"
                            />
                        </>
                    )}

                    <small className="text-muted">
                        By registering, you agree to our{' '}
                        <a href="https://about.sourcegraph.com/terms" target="_blank" rel="noopener">
                            Terms of Service
                        </a>{' '}
                        and{' '}
                        <a href="https://about.sourcegraph.com/privacy" target="_blank" rel="noopener">
                            Privacy Policy
                        </a>
                        .
                    </small>

                    {!isSignupOptimised && (
                        <>
                            <hr className={styles.separator} />

                            <div>
                                Already have an account? <Link to={`/sign-in${location.search}`}>Log in</Link>
                            </div>
                        </>
                    )}
                </div>
            </div>
        </div>
    )
}
