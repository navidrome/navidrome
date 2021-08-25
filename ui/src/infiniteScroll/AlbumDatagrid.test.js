import { rowIndexGenerator } from './AlbumDatagrid'

describe('AlbumDatagrid', () => {
  it('maps grid row to correct data indexes', () => {
    const getIndexes = rowIndexGenerator(3, 10)
    expect(getIndexes(0)).toEqual([0, 1, 2])

    // Last row need not contain the full column
    expect(getIndexes(3)).toEqual([9])

    // Single data item
    const getIndexesForSingleItem = rowIndexGenerator(4, 1)
    expect(getIndexesForSingleItem(0)).toEqual([0])
    expect(getIndexesForSingleItem(1)).toEqual([])
  })
})
