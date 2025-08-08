# mailtoblob MDA

[Config](#config)  
[Usage](#usage)  
[Installation with Postfix](#installation-with-postfix)

A light **message delivery agent** that can use AWS s3 buckets as mailbox storage instead of local storage or physical drive. This can be used with postfix or other MTA (message transfer agent) instead of standard MDA that rely on local disks.

Why use s3 as mail storage?
* unlimited mailbox size
* unlimited number  of mailboxes
* journaling
* feed for forensics / surveillance systems

> This is only MDA with no MRA capabilities, in order to retrieve messages you have to use other means of retrieving bucket objects

The only dependencies are the standard library and aws-sdk-go

To build locally (not system wide) clone this repo cd into dir where main.go is, get all dependencies and build:
```
go get ./...
go build -o mailtoblob
```
## Config
**mailtoblob** uses local config for mapping of user email addresses to s3 buckets, you can also specify CATCH ALL route for all calls without valid email address ( in order to use catch_all do not pass -a / -address flag).

**requestConfig** consists of:
- **region** `string` bucket region
- **timeout** (optional) `int` time out setting for all put requests ( default is 0 )
- **endpoint** (optional) `bool` if **true** all put request will use **s3.&lt;region&gt;.amazonaws.com** ( you cannot use this setting when testing locally )

Each **mailbox** mapping consists of:
- **address** `string` email address that the message is addressed to
- **bucket** `string` bucket name to store the massage in
- **cmkKeyArn** (optional) `string` if specified mailtoblob will use server side encryption before storing the message in the bucket.  
Without the key, message body will be saved in plain text.
- **prefix** (optional) `string` prefix to be added to objects of particular mailbox. Any string can be added and also GO **time.Now()** formatted just like regular date time **layout**  
example: "prefix": "dateTimeFormat(20060102)" object will be saved as 20200811/objectKey
```json
{
    "requestConfig": {
        "region": "eu-west-1",
        "timeout": 10,
        "endpoint": false
    },
    "mailboxes": [
        {
            "address": "user@host.co.uk",
            "bucket": "my_bucket",
            "cmkKeyArn": "arn:aws:kms:eu-west-1:123456:key/123456-123456-123456" // cmkKey is optional
            "prefix": "user"
        },
        {
            "address": "user2@host.co.uk",
            "bucket": "my_bucket2",
            "cmkKeyArn": "arn:aws:kms:eu-west-1:123456:key/123456-123456-123456" // cmkKey is optional
            "prefix": "dateTimeFormat(20060102)"
        }
    ]
}
```
## Usage
To pass message body from MTA to s3: ( this is the same for local testing )
```
mailtoblob -a=user@host.co.uk -f=from@host.co.uk "message body"
// or from pipe
echo "message body" | mailtoblob -a=user@host.co.uk -f=from@host.co.uk
```
Postfix master.cf  
```
mailtoblob  unix  -       n       n       -       -       pipe
      flags=ODRhu user=mailtoblob argv=/usr/local/bin/mailtoblob/mailtoblob -a=${recipient} -f=${sender}
```
this will store message for **user@host.co.uk** in **my_bucket**

## Installation with Postfix

append to the bottom of **/etc/postfix/main.cf**
```
default_privs=mailtoblob
mailtoblob_destination_recipient_limit = 1
virtual_mailbox_domains = <hostname>.co.uk
virtual_transport = mailtoblob
virtual_mailbox_maps = hash:/etc/postfix/virtual_mailbox
```
append to the bottom of **/etc/postfix/master.cf**
```
mailtoblob  unix  -       n       n       -       -       pipe
      flags=ODRhu user=mailtoblob argv=/usr/local/bin/mailtoblob/mailtoblob -a ${recipient}
```
create file **/etc/postfix/virtual_mailbox** and add the recipient addresses for which mailtoblob will be triggered  
in the form of:  
user@domain some_text or @domain catch_all
```
user1@<hostname>.co.uk ( text here does not matter but must be present )
user2@<hostname>.co.uk ( text here does not matter but must be present )
```
Create service user **mailtoblob** this is the user postfix will assume to use mailtoblob, we cannot use postfix user or root.  
Create log file and change permissions to mailtoblob user
```
sudo touch /var/log/mailtoblob.log
sudo chown mailtoblob:mailtoblob /var/log/mailtoblob.log
```
Create folder for the binaries and config, and place your files in there:
```
sudo mkdir /usr/local/bin/mailtoblob
sudo chown mailtoblob:mailtoblob /usr/local/bin/mailtoblob
```
Copy build artifacts to /usr/local/bin/mailtoblob, and create your config.json file in the same dir.
Reload virtual maps and postfix
```
sudo postmap /etc/postfix/virtual_mailbox
sudo service postfix restart
```
