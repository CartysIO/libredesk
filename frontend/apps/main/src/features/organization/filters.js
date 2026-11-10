// organizationConversationFilters returns the conversations/all filter for one organization's conversations.
export function organizationConversationFilters(organizationID) {
  return [
    {
      model: 'conversations',
      field: 'organization_id',
      operator: 'equals',
      value: String(organizationID)
    }
  ]
}
