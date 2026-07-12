package s3store

import "fmt"

// S3 key prefix layout (see asutport-rules.mdc).
const (
	PrefixDocsOriginal = "docs"
	PrefixKBAssets     = "kb"
	PrefixTicketAttach = "tickets"
	PrefixSnapshots    = "snapshots"
	PrefixProjects     = "projects"
)

func DocOriginalKey(manufacturerSlug, productSlug, version, filename string) string {
	return PrefixDocsOriginal + "/" + manufacturerSlug + "/" + productSlug + "/" + version + "/original/" + filename
}

func DocPageKey(manufacturerSlug, productSlug, version string, page int) string {
	return pageKey(PrefixDocsOriginal+"/"+manufacturerSlug+"/"+productSlug+"/"+version+"/pages", page, ".png")
}

func DocParsedKey(manufacturerSlug, productSlug, version string, page int) string {
	return pageKey(PrefixDocsOriginal+"/"+manufacturerSlug+"/"+productSlug+"/"+version+"/parsed", page, ".md")
}

func KBAssetKey(articleID, filename string) string {
	return PrefixKBAssets + "/" + articleID + "/assets/" + filename
}

func TicketAttachmentKey(ticketID, uuid, filename string) string {
	return PrefixTicketAttach + "/" + ticketID + "/attachments/" + uuid + "_" + filename
}

func SnapshotKey(installationID, timestamp string) string {
	return PrefixSnapshots + "/" + installationID + "/" + timestamp + ".json"
}

func ProjectKey(installationID, uuid, filename string) string {
	return PrefixProjects + "/" + installationID + "/" + uuid + "_" + filename
}

func pageKey(prefix string, page int, ext string) string {
	return fmt.Sprintf("%s/%04d%s", prefix, page, ext)
}
