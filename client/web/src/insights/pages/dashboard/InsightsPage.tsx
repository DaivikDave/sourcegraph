import GearIcon from 'mdi-react/GearIcon'
import PlusIcon from 'mdi-react/PlusIcon'
import React, { useCallback, useEffect, useMemo, useContext } from 'react'

import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { Link } from '@sourcegraph/shared/src/components/Link'
import { ExtensionsControllerProps } from '@sourcegraph/shared/src/extensions/controller'
import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { useObservable } from '@sourcegraph/shared/src/util/useObservable'
import { PageHeader } from '@sourcegraph/wildcard'

import { FeedbackBadge } from '../../../components/FeedbackBadge'
import { Page } from '../../../components/Page'
import { InsightsIcon, InsightsViewGrid, InsightsViewGridProps } from '../../components'
import { InsightsApiContext } from '../../core/backend/api-provider'

import { useDeleteInsight } from './hooks/use-delete-insight'

export interface InsightsPageProps
    extends ExtensionsControllerProps,
        Omit<InsightsViewGridProps, 'views'>,
        TelemetryProps,
        PlatformContextProps<'updateSettings'> {
    isCreationUIEnabled: boolean
}

/**
 * Renders insight page. (insights grid and navigation for insight)
 */
export const InsightsPage: React.FunctionComponent<InsightsPageProps> = props => {
    const { isCreationUIEnabled, settingsCascade, platformContext } = props
    const { getInsightCombinedViews } = useContext(InsightsApiContext)

    const views = useObservable(
        useMemo(() => getInsightCombinedViews(props.extensionsController?.extHostAPI), [
            props.extensionsController,
            getInsightCombinedViews,
        ])
    )

    const { handleDelete } = useDeleteInsight({ settingsCascade, platformContext })

    // Tracking handlers and logic
    useEffect(() => {
        props.telemetryService.logViewEvent('Insights')
    }, [props.telemetryService])

    const logConfigureClick = useCallback(() => {
        props.telemetryService.log('InsightConfigureClick')
    }, [props.telemetryService])

    const logAddMoreClick = useCallback(() => {
        props.telemetryService.log('InsightAddMoreClick')
    }, [props.telemetryService])

    return (
        <div className="w-100">
            <Page>
                <PageHeader
                    annotation={<FeedbackBadge status="prototype" feedback={{ mailto: 'support@sourcegraph.com' }} />}
                    path={[{ icon: InsightsIcon, text: 'Code insights' }]}
                    actions={
                        !isCreationUIEnabled ? (
                            <>
                                <Link
                                    to="/extensions?query=category:Insights"
                                    onClick={logAddMoreClick}
                                    className="btn btn-secondary mr-1"
                                >
                                    <PlusIcon className="icon-inline" /> Add more insights
                                </Link>
                                <Link to="/user/settings" onClick={logConfigureClick} className="btn btn-secondary">
                                    <GearIcon className="icon-inline" /> Configure insights
                                </Link>
                            </>
                        ) : (
                            <Link
                                to="/insights/create-intro"
                                onClick={logAddMoreClick}
                                className="btn btn-secondary mr-1"
                            >
                                <PlusIcon className="icon-inline" /> Create new insight
                            </Link>
                        )
                    }
                    className="mb-3"
                />
                {views === undefined ? (
                    <div className="d-flex w-100">
                        <LoadingSpinner className="my-4" />
                    </div>
                ) : (
                    <InsightsViewGrid
                        {...props}
                        views={views}
                        hasContextMenu={isCreationUIEnabled}
                        onDelete={handleDelete}
                    />
                )}
            </Page>
        </div>
    )
}
