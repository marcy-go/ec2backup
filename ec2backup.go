package main

import (
  "fmt"
  "os"
  "strings"
  "strconv"
  "sort"
  "time"
  "net/http"
  "io/ioutil"
  "github.com/awslabs/aws-sdk-go/aws"
  "github.com/awslabs/aws-sdk-go/gen/ec2"
  "github.com/mitchellh/cli"
)

type Self struct{}

func (s *Self) Help() string {
  return "ec2backup own"
}

func (s *Self) Synopsis() string {
  return "Backing up myself."
}

func (s *Self) Run(args []string) int {
  ins_id, err := getInstanceId()
  if err != nil {
    fmt.Fprintln(os.Stderr, err.Error())
    return 1
  }
  err = backup([]string{ins_id})
  if err != nil {
    fmt.Fprintln(os.Stderr, err.Error())
    return 1
  }
  return 0
}

type All struct{}

func (s *All) Help() string {
  return "ec2backup all"
}

func (s *All) Synopsis() string {
  return "Backing up all instances."
}

func (s *All) Run(args []string) int {
  err := backup([]string{})
  if err != nil {
    fmt.Fprintln(os.Stderr, err.Error())
    return 1
  }
  return 0
}

func backup(ids []string) error {
  region, err := getRegion()
  if err != nil {
    return err
  }
  req := ec2.DescribeInstancesRequest{}
  req.InstanceIDs = ids
  ec2_cli := ec2.New(aws.IAMCreds(), region, nil)
  res, err := ec2_cli.DescribeInstances(&req)
  if err != nil {
    return err
  }

  for _, r := range res.Reservations {
    for _, ins := range r.Instances {
      name, gen, err := filEC2Tags(&ins)
      if err != nil {
        fmt.Fprintln(os.Stderr, err.Error())
      }
      if gen > 0 {
        created_at := time.Now().Format("200601021504")
        img_name := fmt.Sprintf("%s-%s", name, created_at)
        img_id, err := createImage(ec2_cli, &ins, img_name)
        fmt.Print("Creating image : ")
        fmt.Print(img_id)
        if err != nil {
          fmt.Fprintln(os.Stderr, err.Error())
        } else {
          fmt.Println(" => Complete")
        }

        del_imgs, err := choiceImages(ec2_cli, &ins, gen)
        if err != nil {
          return err
        }
        for _, del_img := range del_imgs {
          fmt.Print("Deleting old image : ")
          fmt.Print(del_img.Image.ImageID)
          err := deleteImage(ec2_cli, del_img.Image.ImageID)
          if err != nil {
            fmt.Fprintln(os.Stderr, err.Error())
          } else {
            fmt.Println(" => Complete")
          }
          for _, dev_map := range del_img.Image.BlockDeviceMappings {
            fmt.Print("Deleting old snapshot : ")
            fmt.Print(dev_map.EBS.SnapshotID)
            err := deleteSnapshot(ec2_cli, dev_map.EBS.SnapshotID)
            if err != nil {
              fmt.Fprintln(os.Stderr, err.Error())
            } else {
              fmt.Println(" => Complete")
            }
          }
        }
        fmt.Print("Tagging image : ")
        fmt.Print(img_id)
        err = tagImage(ec2_cli, img_id, img_name, created_at, &ins)
        if err != nil {
          fmt.Fprintln(os.Stderr, err.Error())
        } else {
          fmt.Println(" => Complete")
        }
        blocks, err := getBlockDeviceMappings(ec2_cli, img_id)
        if err != nil {
          return err
        }
        for _, block := range blocks {
          fmt.Print("Tagging snapshot : ")
          fmt.Print(block.EBS.SnapshotID)
          err := tagSnapshot(ec2_cli, fmt.Sprint(block.EBS.SnapshotID), img_name, fmt.Sprint(block.DeviceName), created_at, &ins)
          if err != nil {
            fmt.Fprintln(os.Stderr, err.Error())
          } else {
            fmt.Println(" => Complete")
          }
        }
      }
    }
  }
  return nil
}

func getBlockDeviceMappings(ec2_cli *ec2.EC2, img_id string) ([]ec2.BlockDeviceMapping, error) {
  var req ec2.DescribeImagesRequest
  req.ImageIDs = []string{img_id}
  res, err := ec2_cli.DescribeImages(&req)
  if err != nil {
    return []ec2.BlockDeviceMapping{}, err
  }
  return res.Images[0].BlockDeviceMappings, nil
}

func tagImage(ec2_cli *ec2.EC2, id, name, at string, ins *ec2.Instance) error {
  var req ec2.CreateTagsRequest
  req.Resources = []string{id}
  var tag1, tag2, tag3, tag4 ec2.Tag
  tag1.Key   = aws.String("Name")
  tag1.Value = aws.String(name)
  tag2.Key   = aws.String("InstanceID")
  tag2.Value = ins.InstanceID
  tag3.Key   = aws.String("CreatedAt")
  tag3.Value = aws.String(at)
  tag4.Key   = aws.String("Backup-Type")
  tag4.Value = aws.String("auto")
  req.Tags = []ec2.Tag{tag1, tag2, tag3, tag4}
  return ec2_cli.CreateTags(&req)
}

func tagSnapshot(ec2_cli *ec2.EC2, id, name, dev, at string, ins *ec2.Instance) error {
  var req ec2.CreateTagsRequest
  req.Resources = []string{id}
  var tag1, tag2, tag3, tag4 ec2.Tag
  tag1.Key   = aws.String("Name")
  tag1.Value = aws.String(fmt.Sprintf("%s-%s", name, dev))
  tag2.Key   = aws.String("InstanceID")
  tag2.Value = ins.InstanceID
  tag3.Key   = aws.String("CreatedAt")
  tag3.Value = aws.String(at)
  tag4.Key   = aws.String("Backup-Type")
  tag4.Value = aws.String("auto")
  req.Tags = []ec2.Tag{tag1, tag2, tag3, tag4}
  return ec2_cli.CreateTags(&req)
}

func deleteImage(ec2_cli *ec2.EC2, id aws.StringValue) error {
  var req ec2.DeregisterImageRequest
  req.ImageID = id
  return ec2_cli.DeregisterImage(&req)
}

func deleteSnapshot(ec2_cli *ec2.EC2, id aws.StringValue) error {
  var req ec2.DeleteSnapshotRequest
  req.SnapshotID = id
  return ec2_cli.DeleteSnapshot(&req)
}

type SortImage struct {
  CreatedAt string
  Image ec2.Image
}

type SortImages []SortImage

func (img SortImages) Len() int {
    return len(img)
}

func (img SortImages) Swap(i, j int) {
  img[i], img[j] = img[j], img[i]
}

func (img SortImages) Less(i, j int) bool {
    return img[i].CreatedAt < img[j].CreatedAt
}

func choiceImages(ec2_cli *ec2.EC2, ins *ec2.Instance, gen int) (SortImages, error) {
  var req ec2.DescribeImagesRequest
  var fil1, fil2 ec2.Filter
  fil1.Name  = aws.String("tag:InstanceID")
  fil1.Values = []string{fmt.Sprint(ins.InstanceID)}
  fil2.Name  = aws.String("tag:Backup-Type")
  fil2.Values = []string{"auto"}
  req.Filters = []ec2.Filter{fil1, fil2}
  res, err := ec2_cli.DescribeImages(&req)
  var imgs SortImages
  if err != nil {
    return imgs, err
  }
  sort_imgs := make(SortImages, gen)
  for _, img := range res.Images {
    var sort_img SortImage
    for _, tag := range img.Tags {
      if tag.Key == aws.String("CreatedAt") {
        sort_img.CreatedAt = fmt.Sprint(tag.Value)
        sort_img.Image = img
        sort_imgs = append(sort_imgs, sort_img)
      }
    }
  }
  sort.Sort(sort_imgs)
  return sort_imgs[:gen-1], nil
}

func createImage(ec2_cli *ec2.EC2, ins *ec2.Instance, img_name string) (string, error) {
  var req ec2.CreateImageRequest
//  req.BlockDeviceMappings = ins.BlockDeviceMappings
  req.InstanceID          = ins.InstanceID
  req.Name                = aws.String(img_name)
  req.Description         = aws.String(fmt.Sprintf("Created from %s at %s", ins.InstanceID, time.Now().Format("2006-01-02 15:04")))
  req.NoReboot            = aws.Boolean(true)
  ret, err := ec2_cli.CreateImage(&req)
  return fmt.Sprint(ret.ImageID), err
}

func filEC2Tags(ins *ec2.Instance) (string, int, error) {
  name := ""
  gen  := 0
  var err error
  for _, tag := range ins.Tags {
    key := strings.Trim(" ", fmt.Sprint(tag.Key))
    val := strings.Trim(" ", fmt.Sprint(tag.Value))
    if key == "Name" {
      name = fmt.Sprint(tag.Value)
    }
    if val == "Backup-Generation" {
      gen, err = strconv.Atoi(val)
    }
  }
  return name, gen, err
}

func getMetaData(url string) (string, error) {
  res, err := http.Get("http://169.254.169.254/latest/meta-data" + url)
  defer res.Body.Close()
  if err != nil {
    return "", err
  }
  body, err := ioutil.ReadAll(res.Body)
  if err != nil {
    return "", err
  }
  return string(body), nil
}

func getInstanceId() (string, error) {
  return getMetaData("/instance-id")
}

func getRegion() (string, error) {
  meta, err := getMetaData("/placement/availability-zone")
  if err != nil {
    return "", err
  }
  end  := len(meta) - 1
  return meta[0:end], nil
}

func main() {
  c := cli.NewCLI("ec2backup", "0.0.2")
  c.Args = os.Args[1:]
  c.Commands = map[string]cli.CommandFactory{
    "self": func() (cli.Command, error) {
      return &Self{}, nil
    },
    "all": func() (cli.Command, error) {
      return &All{}, nil
    },
  }
  ret, err := c.Run()
  if err != nil {
    fmt.Fprintf(os.Stderr, err.Error())
  }
  os.Exit(ret)
}
