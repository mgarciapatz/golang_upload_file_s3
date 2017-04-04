package controllers
import(
	"io"
	"os"
	"fmt"
	"time"
	"bytes"
	//"strings"
	"strconv"
	"net/http"
	"path/filepath"
	"erp-api/helper"
	"github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/service/s3"
    "github.com/aws/aws-sdk-go/aws/session"
    _"github.com/aws/aws-sdk-go/aws/awsutil"
    "github.com/aws/aws-sdk-go/aws/credentials"
)
func AwsConnectionS3 () (*aws.Config, interface{}) {
	aws_access_key_id		:=	"" //aws access key
	aws_secret_access_key	:=	"" //aws secret key
	token					:=	""
	creds := credentials.NewStaticCredentials(aws_access_key_id, aws_secret_access_key, token)
	_, err := creds.Get()
	if err != nil {
		return nil, err
	}
	cfg := aws.NewConfig().WithRegion("ap-southeast-1").WithCredentials(creds)
	return cfg, nil
}

func AWSFileUploadS3(w http.ResponseWriter, r *http.Request,next http.HandlerFunc) {
	var isOk 	  bool
	var isS3Ok	  bool
	var temp_file string
	var nameList  []string

	read, err := r.MultipartReader()
	if err != nil {
		respondJson(err,rescode.INVALID,w)
		return
	}else {
		for {
			part, readErr := read.NextPart()
	        if readErr == io.EOF {
	        	break
	        }else {
		        if part.FormName() == "image" {
		        	temp_file = "/var/www/html/erp-front-end/temp_file/"
			        isExist,dirErr := helper.PathExist(temp_file)
			        if dirErr == nil {
			            if !isExist {
			                os.Mkdir(temp_file,0777)
			            }
			        }
			        newFile 	:= strconv.FormatInt(time.Now().UTC().UnixNano(), 10) + filepath.Ext(part.FileName())
			        nameList 	 = append(nameList, newFile)
			        dst, dstErr := os.Create(temp_file + newFile)
			        if dstErr != nil {
			            break
			        }else {
			            fileSize, ioErr := io.Copy(dst, part)
			            if ioErr != nil {
			                break
			            }
			            if fileSize > 5000000 {
			                os.Remove(temp_file+newFile)
			                break
			            }
			        	defer dst.Close()
			        	isOk = true
			        }
		        }
	        }

		}
		if isOk {
			connection, connectErr := AwsConnectionS3()
			fmt.Println(connection, connectErr)
			if connectErr != nil {
				respondJson("Connection to AWS S3 Failed", rescode.INVALID,w)
				return
			}else {
				for _, image := range nameList {
					file, errFile := os.Open(temp_file + image)
					defer file.Close()
					if errFile != nil {
						respondJson("Unknown file content", rescode.INVALID,w)
						return
					}else {
						fileInfo, _ := file.Stat()
						size 		:= fileInfo.Size()
						buffer 		:= make([]byte, size)
						file.Read(buffer)
						fileBytes 	:= bytes.NewReader(buffer)
						fileType    := http.DetectContentType(buffer)
						
						service	:= s3.New(session.New(), connection)
						params := &s3.PutObjectInput{
							Bucket			:	aws.String("erp-fileupload-golang"),
							Key				:	aws.String("/announcements/images/"+image),
							Body			:	fileBytes,
							ContentLength	:	aws.Int64(size),
							ContentType		:	aws.String(fileType),
							ACL				:	aws.String("public-read"),
						}

						_ , respErr	:=	service.PutObject(params)
				    	if respErr != nil {
				    		respondJson("Failed to upload on AWS S3", rescode.INVALID,w)
							return
				    	}else {
				    		os.Remove(temp_file+image)
				    		isS3Ok = true
				    	}

					}
				}
			}
		}
		if isS3Ok {
			respondJson("File uploaded successfully", rescode.OK, w)
			return
		}
	}
}