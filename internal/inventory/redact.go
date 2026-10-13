package inventory

// Audience is resolved by the service, never supplied by a client.
type Audience struct{ Sensitive, Guest bool }

func redactFields(fields []Field, audience Audience) []Field {
	out := make([]Field, 0, len(fields))
	for _, field := range fields {
		if field.Sensitive && !audience.Sensitive {
			if audience.Guest {
				continue
			}
			field.Value = nil
			field.Redacted = true
		}
		out = append(out, field)
	}
	return out
}

func redactNode(node *Node, audience Audience) {
	node.Properties = redactFields(node.Properties, audience)
	if audience.Guest {
		node.Health.Message = nil
		node.CreatedBy = nil
	}
}

func redactLink(link *Link, audience Audience) {
	link.Properties = redactFields(link.Properties, audience)
	if audience.Guest {
		link.Health.Message = nil
	}
}

func redactArea(area *Area, audience Audience) {
	area.Properties = redactFields(area.Properties, audience)
	if !audience.Sensitive {
		area.CIDR = nil
		area.VlanID = nil
	}
}

func redactNote(note *Note, audience Audience) bool {
	if note.IsSensitive && !audience.Sensitive {
		if audience.Guest {
			return false
		}
		note.Title = ""
		note.Body = ""
		note.Redacted = true
	}
	if audience.Guest {
		note.CreatedBy = nil
		note.UpdatedBy = nil
	}
	return true
}
