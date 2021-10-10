import classNames from 'classnames'
import * as H from 'history'
import GithubIcon from 'mdi-react/GithubIcon'
import * as React from 'react'

import { LoaderInput } from '@sourcegraph/branded/src/components/LoaderInput'
import { deriveInputClassName, InputValidationState } from '@sourcegraph/shared/src/util/useInputValidation'

import { SourcegraphContext } from '../jscontext'
import { USERNAME_MAX_LENGTH, VALID_USERNAME_REGEXP } from '../user'

import styles from './CloudSignUpPage.module.scss'

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
    inputRef?: React.Ref<HTMLInputElement>
}

interface SignupEmailField {
    emailState: InputValidationState
    loading: boolean
    label: string
    nextEmailFieldChange: (changeEvent: React.ChangeEvent<HTMLInputElement>) => void
    emailInputReference: React.Ref<HTMLInputElement>
}

interface ExternalsAuthProps {
    context: Pick<SourcegraphContext, 'authProviders' | 'experimentalFeatures'>
    githubLabel: string
    gitlabLabel: string
    onClick: (type: string) => void
    withCenteredText?: boolean
}

export const PasswordInput: React.FunctionComponent<InputProps> = props => {
    const { inputRef, ...other } = props
    return (
        <input
            name="password"
            id="password"
            {...other}
            className={classNames('form-control', props.className)}
            placeholder={props.placeholder || 'Password'}
            type="password"
            required={true}
            ref={inputRef}
        />
    )
}

export const EmailInput: React.FunctionComponent<InputProps> = props => {
    const { inputRef, ...other } = props
    return (
        <input
            name="email"
            id="email"
            {...other}
            className={classNames('form-control', props.className)}
            type="email"
            placeholder={props.placeholder || 'Email'}
            spellCheck={false}
            autoComplete="email"
            ref={inputRef}
        />
    )
}

export const UsernameInput: React.FunctionComponent<InputProps> = props => {
    const { inputRef, ...other } = props
    return (
        <input
            name="username"
            id="username"
            {...other}
            className={classNames('form-control', props.className)}
            type="text"
            placeholder={props.placeholder || 'Username'}
            spellCheck={false}
            pattern={VALID_USERNAME_REGEXP}
            maxLength={USERNAME_MAX_LENGTH}
            autoCapitalize="off"
            autoComplete="username"
            ref={inputRef}
        />
    )
}

export const SignupEmailField: React.FunctionComponent<SignupEmailField> = ({
    emailState,
    loading,
    label,
    nextEmailFieldChange,
    emailInputReference,
}) => (
    <div className="form-group d-flex flex-column align-content-start">
        <label
            htmlFor="email"
            className={classNames('align-self-start', {
                'text-danger font-weight-bold': emailState.kind === 'INVALID',
            })}
        >
            {label}
        </label>
        <LoaderInput className={classNames(deriveInputClassName(emailState))} loading={emailState.kind === 'LOADING'}>
            <EmailInput
                className={deriveInputClassName(emailState)}
                onChange={nextEmailFieldChange}
                required={true}
                value={emailState.value}
                disabled={loading}
                autoFocus={true}
                placeholder=" "
                inputRef={emailInputReference}
            />
        </LoaderInput>
        {emailState.kind === 'INVALID' && <small className="invalid-feedback">{emailState.reason}</small>}
    </div>
)

const GitlabColorIcon: React.FunctionComponent<{ className?: string }> = ({ className }) => (
    <svg
        className={className}
        width="24"
        height="24"
        viewBox="-2 -2 26 26"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
    >
        <path d="M9.99944 19.2025L13.684 7.86902H6.32031L9.99944 19.2025Z" fill="#E24329" />
        <path
            d="M1.1594 7.8689L0.037381 11.3121C-0.0641521 11.6248 0.0454967 11.9699 0.313487 12.1648L9.99935 19.2023L1.1594 7.8689Z"
            fill="#FCA326"
        />
        <path
            d="M1.15918 7.86873H6.31995L4.0989 1.04315C3.98522 0.693949 3.48982 0.693949 3.37206 1.04315L1.15918 7.86873Z"
            fill="#E24329"
        />
        <path
            d="M18.8444 7.8689L19.9624 11.3121C20.0639 11.6248 19.9542 11.9699 19.6862 12.1648L9.99902 19.2023L18.8444 7.8689Z"
            fill="#FCA326"
        />
        <path
            d="M18.8449 7.86873H13.6841L15.901 1.04315C16.0147 0.693949 16.5101 0.693949 16.6279 1.04315L18.8449 7.86873Z"
            fill="#E24329"
        />
        <path d="M9.99902 19.2023L13.6835 7.8689H18.8444L9.99902 19.2023Z" fill="#FC6D26" />
        <path d="M9.99907 19.2023L1.15918 7.8689H6.31995L9.99907 19.2023Z" fill="#FC6D26" />
    </svg>
)

export const ExternalsAuth: React.FunctionComponent<ExternalsAuthProps> = ({
    context,
    githubLabel,
    gitlabLabel,
    onClick,
    withCenteredText,
}) => {
    // Since this component is only intended for use on Sourcegraph.com, it's OK to hardcode
    // GitHub and GitLab auth providers here as they are the only ones used on Sourcegraph.com.
    // In the future if this page is intended for use in Sourcegraph Sever, this would need to be generalized
    // for other auth providers such SAML, OpenID, Okta, Azure AD, etc.

    const githubProvider = context.authProviders.find(provider =>
        provider.authenticationURL?.startsWith('/.auth/github/login?pc=https%3A%2F%2Fgithub.com%2F')
    )
    const gitlabProvider = context.authProviders.find(provider =>
        provider.authenticationURL?.startsWith('/.auth/gitlab/login?pc=https%3A%2F%2Fgitlab.com%2F')
    )

    return (
        <>
            {githubProvider && (
                <a
                    href={maybeAddPostSignUpRedirect(githubProvider.authenticationURL)}
                    className={classNames(
                        withCenteredText ? 'd-flex justify-content-center' : '',
                        'text-decoration-none',
                        styles.signUpButton,
                        styles.githubButton
                    )}
                    onClick={() => onClick('github')}
                >
                    <GithubIcon className="mr-3" /> {githubLabel}
                </a>
            )}

            {gitlabProvider && (
                <a
                    href={maybeAddPostSignUpRedirect(gitlabProvider.authenticationURL)}
                    className={classNames(
                        withCenteredText ? 'd-flex justify-content-center' : '',
                        'text-decoration-none',
                        styles.signUpButton,
                        styles.gitlabButton
                    )}
                    onClick={() => onClick('gitlab')}
                >
                    <GitlabColorIcon className="mr-3" /> {gitlabLabel}
                </a>
            )}
        </>
    )
}

/**
 * Returns the sanitized return-to relative URL (including only the path, search, and fragment).
 * This is the location that a user should be returned to after performing signin or signup to continue
 * to the page they intended to view as an authenticated user.
 *
 * 🚨 SECURITY: We must disallow open redirects (to arbitrary hosts).
 */
export function getReturnTo(location: H.Location): string {
    const searchParameters = new URLSearchParams(location.search)
    const returnTo = searchParameters.get('returnTo') || '/search'
    const newURL = new URL(returnTo, window.location.href)

    newURL.searchParams.append('toast', 'integrations')
    return newURL.pathname + newURL.search + newURL.hash
}

export function maybeAddPostSignUpRedirect(url?: string): string {
    const enablePostSignupFlow = window.context?.experimentalFeatures?.enablePostSignupFlow
    const isDotCom = window.context?.sourcegraphDotComMode
    const shouldAddRedirect = isDotCom && enablePostSignupFlow

    if (url) {
        if (shouldAddRedirect) {
            // second param to protect against relative urls
            const urlObject = new URL(url, window.location.href)

            urlObject.searchParams.append('redirect', '/welcome')
            return urlObject.toString()
        }

        return url
    }

    return shouldAddRedirect ? '/welcome' : ''
}
