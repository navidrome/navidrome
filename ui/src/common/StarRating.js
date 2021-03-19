import React from 'react'
import StarIcon from '@material-ui/icons/Star'
import { makeStyles } from '@material-ui/core/styles'
import { useStarRating } from "./useStarRating";

const useStyles = makeStyles({
    rated: {
        color : '#ffc107'
    },
    unrated: {
        color : '#e4e5e9'
    },
    input: {
        display: 'none'
    },
    star : {
        cursor: 'pointer',
        transition: 'color 200ms' 
    }
})


export const StarRating = ({record = {}, source, resource}) => {
    const [ rate, hoverRating, hover, rating] = useStarRating(resource, record, source)
    const classes = useStyles()

    return (
        <div>
            {[...Array(5)].map((star, i) => {
                const ratingVal = i + 1

                return (
                    <span key={i}>
                        <input 
                            style={{display: 'none'}}
                            type="radio" 
                            name="rating" 
                            value={ratingVal > 0 ? true : false}
                            onClick={() => rate(ratingVal)} 
                        />
                        <StarIcon 
                            className={ratingVal <= (hover || rating) ? classes.rated : classes.unrated + " " + classes.star }
                            onMouseEnter={() => hoverRating(ratingVal)}
                            onMouseLeave={() => hoverRating(null)} 
                        />
                    </span>

                )
            })}
        </div>
    )
}