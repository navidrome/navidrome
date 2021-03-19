import { useState, useCallback } from 'react'
import { useDataProvider, useNotify } from 'react-admin'
import subsonic from "../subsonic"

export const useStarRating = (resource, record = {}, source) => {
    const [starRating, setStarRating] = useState(record[source])
    const [hover, setHover] = useState(null)
    const notify = useNotify()
    const dataProvider = useDataProvider()

    const refreshRating = useCallback(() => {
        dataProvider.getOne(resource, {id: record.id})
        .then(data => {
            setStarRating(data.rating)
        })
    }, [dataProvider, record.id, resource])


    const rate = (val) => {
        subsonic.setRating(record.id, val)
        .then(refreshRating)
        .catch((e) => {
            console.log('Error setting star rating: ', e)
            notify('ra.page.error', 'warning')  
        })
    }

    const hoverRating = (val) => {
        setHover(val)
    }

    return [rate, hoverRating, hover, starRating]
}

