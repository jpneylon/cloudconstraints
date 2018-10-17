package main

import (
	"fmt"
	"net/http"
	"bytes"
	"io"
	"io/ioutil"
	"strings"

	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/appengine"
	"google.golang.org/appengine/file"
	"google.golang.org/appengine/log"
)

type buckread struct {
	client 		*storage.Client
	bucketName	string
	bucket		*storage.BucketHandle
	
	w 		io.Writer
	ctx 		context.Context
	cleanUp		[]string
	failed 		bool
}

func main() {
	http.HandleFunc("/", handle)
	appengine.Main()
}

func handle(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Welcome to Cloud Constraints. >:(--|--{ \n")
	
	ctx := appengine.NewContext(r)

	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	//[START get_default_bucket]
	// Use `dev_appserver.py --default_gcs_bucket_name GCS_BUCKET_NAME`
	// when running locally.
	bucket, err := file.DefaultBucketName(ctx)
	fmt.Fprintln(w, "Buckethead: %v \n",bucket)
	if err != nil {
		log.Errorf(ctx, "failed to get default GCS bucket name: %v", err)
	}
	//[END get_default_bucket]

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Errorf(ctx, "failed to create client: %v", err)
		return
	}
	defer client.Close()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Demo GCS Application running from Version: %v\n", appengine.VersionID(ctx))
	fmt.Fprintf(w, "Using bucket name: %v\n\n", bucket)
	fmt.Fprintln("Demo GCS Application running from Version: %v\n", appengine.VersionID(ctx))
	fmt.Fprintln("Using bucket name: %v\n\n", bucket)

	buf := &bytes.Buffer{}
	d := &buckread{
		w:          buf,
		ctx:        ctx,
		client:     client,
		bucket:     client.Bucket(bucket),
		bucketName: bucket,
	}

	n := "data/TG101_Constraints.csv"
	//d.createFile(n)
	d.readFile(n)
	//d.copyFile(n)
	d.statFile(n)
	d.createListFiles()
	d.listBucket()
	d.listBucketDirMode()
	//d.defaultACL()
	//d.putDefaultACLRule()
	//d.deleteDefaultACLRule()
	//d.bucketACL()
	//d.putBucketACLRule()
	//d.deleteBucketACLRule()
	//d.acl(n)
	//d.putACLRule(n)
	//d.deleteACLRule(n)
	//d.deleteFiles()

	if d.failed {
		w.WriteHeader(http.StatusInternalServerError)
		buf.WriteTo(w)
		fmt.Fprintf(w, "\n Buckread failed. \n")
	} else {
		w.WriteHeader(http.StatusOK)
		buf.WriteTo(w)
		fmt.Fprintf(w, "\n Buckread succeeded. \n")
	}
}




func (d *buckread) errorf(format string, args ...interface{}) {
	d.failed = true
	fmt.Fprintln(d.w, fmt.Sprintf(format, args...))
	log.Errorf(d.ctx, format, args...)
}

//[START write]
// createFile creates a file in Google Cloud Storage.
func (d *buckread) createFile(fileName string) {
	fmt.Fprintf(d.w, "Creating file /%v/%v\n", d.bucketName, fileName)

	wc := d.bucket.Object(fileName).NewWriter(d.ctx)
	wc.ContentType = "text/plain"
	wc.Metadata = map[string]string{
		"x-goog-meta-foo": "foo",
		"x-goog-meta-bar": "bar",
	}
	d.cleanUp = append(d.cleanUp, fileName)

	if _, err := wc.Write([]byte("abcde\n")); err != nil {
		d.errorf("createFile: unable to write data to bucket %q, file %q: %v", d.bucketName, fileName, err)
		return
	}
	if _, err := wc.Write([]byte(strings.Repeat("f", 1024*4) + "\n")); err != nil {
		d.errorf("createFile: unable to write data to bucket %q, file %q: %v", d.bucketName, fileName, err)
		return
	}
	if err := wc.Close(); err != nil {
		d.errorf("createFile: unable to close bucket %q, file %q: %v", d.bucketName, fileName, err)
		return
	}
}

//[END write]

//[START read]
// readFile reads the named file in Google Cloud Storage.
func (d *buckread) readFile(fileName string) {
	io.WriteString(d.w, "\nAbbreviated file content (first line and last 1K):\n")

	rc, err := d.bucket.Object(fileName).NewReader(d.ctx)
	if err != nil {
		d.errorf("readFile: unable to open file from bucket %q, file %q: %v", d.bucketName, fileName, err)
		return
	}
	defer rc.Close()
	slurp, err := ioutil.ReadAll(rc)
	if err != nil {
		d.errorf("readFile: unable to read data from bucket %q, file %q: %v", d.bucketName, fileName, err)
		return
	}

	fmt.Fprintf(d.w, "%s\n", bytes.SplitN(slurp, []byte("\n"), 2)[0])
	fmt.Fprintln("%s\n", bytes.SplitN(slurp, []byte("\n"), 2)[0])
	if len(slurp) > 1024 {
		fmt.Fprintf(d.w, "...%s\n", slurp[len(slurp)-1024:])
		fmt.Fprintln("...%s\n", slurp[len(slurp)-1024:])
	} else {
		fmt.Fprintf(d.w, "%s\n", slurp)
		fmt.Fprintln("%s\n", slurp)
	}
}

//[END read]

//[START copy]
// copyFile copies a file in Google Cloud Storage.
func (d *buckread) copyFile(fileName string) {
	copyName := fileName + "-copy"
	fmt.Fprintf(d.w, "Copying file /%v/%v to /%v/%v:\n", d.bucketName, fileName, d.bucketName, copyName)

	obj, err := d.bucket.Object(copyName).CopierFrom(d.bucket.Object(fileName)).Run(d.ctx)
	if err != nil {
		d.errorf("copyFile: unable to copy /%v/%v to bucket %q, file %q: %v", d.bucketName, fileName, d.bucketName, copyName, err)
		return
	}
	d.cleanUp = append(d.cleanUp, copyName)

	d.dumpStats(obj)
}

//[END copy]

func (d *buckread) dumpStats(obj *storage.ObjectAttrs) {
	fmt.Fprintf(d.w, "(filename: /%v/%v, ", obj.Bucket, obj.Name)
	fmt.Fprintf(d.w, "ContentType: %q, ", obj.ContentType)
	fmt.Fprintf(d.w, "ACL: %#v, ", obj.ACL)
	fmt.Fprintf(d.w, "Owner: %v, ", obj.Owner)
	fmt.Fprintf(d.w, "ContentEncoding: %q, ", obj.ContentEncoding)
	fmt.Fprintf(d.w, "Size: %v, ", obj.Size)
	fmt.Fprintf(d.w, "MD5: %q, ", obj.MD5)
	fmt.Fprintf(d.w, "CRC32C: %q, ", obj.CRC32C)
	fmt.Fprintf(d.w, "Metadata: %#v, ", obj.Metadata)
	fmt.Fprintf(d.w, "MediaLink: %q, ", obj.MediaLink)
	fmt.Fprintf(d.w, "StorageClass: %q, ", obj.StorageClass)
	if !obj.Deleted.IsZero() {
		fmt.Fprintf(d.w, "Deleted: %v, ", obj.Deleted)
	}
	fmt.Fprintf(d.w, "Updated: %v)\n", obj.Updated)
}

//[START file_metadata]
// statFile reads the stats of the named file in Google Cloud Storage.
func (d *buckread) statFile(fileName string) {
	io.WriteString(d.w, "\nFile stat:\n")

	obj, err := d.bucket.Object(fileName).Attrs(d.ctx)
	if err != nil {
		d.errorf("statFile: unable to stat file from bucket %q, file %q: %v", d.bucketName, fileName, err)
		return
	}

	d.dumpStats(obj)
}

//[END file_metadata]

// createListFiles creates files that will be used by listBucket.
func (d *buckread) createListFiles() {
	io.WriteString(d.w, "\nCreating more files for listbucket...\n")
	for _, n := range []string{"foo1", "foo2", "bar", "bar/1", "bar/2", "boo/"} {
		d.createFile(n)
	}
}

//[START list_bucket]
// listBucket lists the contents of a bucket in Google Cloud Storage.
func (d *buckread) listBucket() {
	io.WriteString(d.w, "\nListbucket result:\n")

	query := &storage.Query{Prefix: "foo"}
	it := d.bucket.Objects(d.ctx, query)
	for {
		obj, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			d.errorf("listBucket: unable to list bucket %q: %v", d.bucketName, err)
			return
		}
		d.dumpStats(obj)
	}
}

//[END list_bucket]

func (d *buckread) listDir(name, indent string) {
	query := &storage.Query{Prefix: name, Delimiter: "/"}
	it := d.bucket.Objects(d.ctx, query)
	for {
		obj, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			d.errorf("listBucketDirMode: unable to list bucket %q: %v", d.bucketName, err)
			return
		}
		if obj.Prefix == "" {
			fmt.Fprint(d.w, indent)
			d.dumpStats(obj)
			continue
		}
		fmt.Fprintf(d.w, "%v(directory: /%v/%v)\n", indent, d.bucketName, obj.Prefix)
		d.listDir(obj.Prefix, indent+"  ")
	}
}

// listBucketDirMode lists the contents of a bucket in dir mode in Google Cloud Storage.
func (d *buckread) listBucketDirMode() {
	io.WriteString(d.w, "\nListbucket directory mode result:\n")
	d.listDir("b", "")
}

// dumpDefaultACL prints out the default object ACL for this bucket.
func (d *buckread) dumpDefaultACL() {
	acl, err := d.bucket.ACL().List(d.ctx)
	if err != nil {
		d.errorf("defaultACL: unable to list default object ACL for bucket %q: %v", d.bucketName, err)
		return
	}
	for _, v := range acl {
		fmt.Fprintf(d.w, "Scope: %q, Permission: %q\n", v.Entity, v.Role)
	}
}

// defaultACL displays the default object ACL for this bucket.
func (d *buckread) defaultACL() {
	io.WriteString(d.w, "\nDefault object ACL:\n")
	d.dumpDefaultACL()
}

// putDefaultACLRule adds the "allUsers" default object ACL rule for this bucket.
func (d *buckread) putDefaultACLRule() {
	io.WriteString(d.w, "\nPut Default object ACL Rule:\n")
	err := d.bucket.DefaultObjectACL().Set(d.ctx, storage.AllUsers, storage.RoleReader)
	if err != nil {
		d.errorf("putDefaultACLRule: unable to save default object ACL rule for bucket %q: %v", d.bucketName, err)
		return
	}
	d.dumpDefaultACL()
}

// deleteDefaultACLRule deleted the "allUsers" default object ACL rule for this bucket.
func (d *buckread) deleteDefaultACLRule() {
	io.WriteString(d.w, "\nDelete Default object ACL Rule:\n")
	err := d.bucket.DefaultObjectACL().Delete(d.ctx, storage.AllUsers)
	if err != nil {
		d.errorf("deleteDefaultACLRule: unable to delete default object ACL rule for bucket %q: %v", d.bucketName, err)
		return
	}
	d.dumpDefaultACL()
}

// dumpBucketACL prints out the bucket ACL.
func (d *buckread) dumpBucketACL() {
	acl, err := d.bucket.ACL().List(d.ctx)
	if err != nil {
		d.errorf("dumpBucketACL: unable to list bucket ACL for bucket %q: %v", d.bucketName, err)
		return
	}
	for _, v := range acl {
		fmt.Fprintf(d.w, "Scope: %q, Permission: %q\n", v.Entity, v.Role)
	}
}

// bucketACL displays the bucket ACL for this bucket.
func (d *buckread) bucketACL() {
	io.WriteString(d.w, "\nBucket ACL:\n")
	d.dumpBucketACL()
}

// putBucketACLRule adds the "allUsers" bucket ACL rule for this bucket.
func (d *buckread) putBucketACLRule() {
	io.WriteString(d.w, "\nPut Bucket ACL Rule:\n")
	err := d.bucket.ACL().Set(d.ctx, storage.AllUsers, storage.RoleReader)
	if err != nil {
		d.errorf("putBucketACLRule: unable to save bucket ACL rule for bucket %q: %v", d.bucketName, err)
		return
	}
	d.dumpBucketACL()
}

// deleteBucketACLRule deleted the "allUsers" bucket ACL rule for this bucket.
func (d *buckread) deleteBucketACLRule() {
	io.WriteString(d.w, "\nDelete Bucket ACL Rule:\n")
	err := d.bucket.ACL().Delete(d.ctx, storage.AllUsers)
	if err != nil {
		d.errorf("deleteBucketACLRule: unable to delete bucket ACL rule for bucket %q: %v", d.bucketName, err)
		return
	}
	d.dumpBucketACL()
}

// dumpACL prints out the ACL of the named file.
func (d *buckread) dumpACL(fileName string) {
	acl, err := d.bucket.Object(fileName).ACL().List(d.ctx)
	if err != nil {
		d.errorf("dumpACL: unable to list file ACL for bucket %q, file %q: %v", d.bucketName, fileName, err)
		return
	}
	for _, v := range acl {
		fmt.Fprintf(d.w, "Scope: %q, Permission: %q\n", v.Entity, v.Role)
	}
}

// acl displays the ACL for the named file.
func (d *buckread) acl(fileName string) {
	fmt.Fprintf(d.w, "\nACL for file %v:\n", fileName)
	d.dumpACL(fileName)
}

// putACLRule adds the "allUsers" ACL rule for the named file.
func (d *buckread) putACLRule(fileName string) {
	fmt.Fprintf(d.w, "\nPut ACL rule for file %v:\n", fileName)
	err := d.bucket.Object(fileName).ACL().Set(d.ctx, storage.AllUsers, storage.RoleReader)
	if err != nil {
		d.errorf("putACLRule: unable to save ACL rule for bucket %q, file %q: %v", d.bucketName, fileName, err)
		return
	}
	d.dumpACL(fileName)
}

// deleteACLRule deleted the "allUsers" ACL rule for the named file.
func (d *demo) deleteACLRule(fileName string) {
	fmt.Fprintf(d.w, "\nDelete ACL rule for file %v:\n", fileName)
	err := d.bucket.Object(fileName).ACL().Delete(d.ctx, storage.AllUsers)
	if err != nil {
		d.errorf("deleteACLRule: unable to delete ACL rule for bucket %q, file %q: %v", d.bucketName, fileName, err)
		return
	}
	d.dumpACL(fileName)
}

// deleteFiles deletes all the temporary files from a bucket created by this demo.
func (d *buckread) deleteFiles() {
	io.WriteString(d.w, "\nDeleting files...\n")
	for _, v := range d.cleanUp {
		fmt.Fprintf(d.w, "Deleting file %v\n", v)
		if err := d.bucket.Object(v).Delete(d.ctx); err != nil {
			d.errorf("deleteFiles: unable to delete bucket %q, file %q: %v", d.bucketName, v, err)
			return
		}
	}
}

//[END sample]
