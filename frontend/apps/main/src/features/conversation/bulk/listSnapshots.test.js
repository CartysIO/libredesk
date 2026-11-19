// @vitest-environment node
import { describe, expect, it } from 'vitest'
import { captureListSnapshots, restoreListSnapshots } from './listSnapshots'

describe('listSnapshots', () => {
  it('restores rows at their original indices after removal', () => {
    const data = [
      { uuid: 'a', status: 'Open' },
      { uuid: 'b', status: 'Open' },
      { uuid: 'c', status: 'Open' }
    ]
    const snapshots = captureListSnapshots(data, ['b'])
    const without = data.filter((row) => row.uuid !== 'b')
    const { data: restored } = restoreListSnapshots(without, 2, snapshots)
    expect(restored.map((c) => c.uuid)).toEqual(['a', 'b', 'c'])
  })
})